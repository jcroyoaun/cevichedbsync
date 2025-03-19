package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cevichev1alpha1 "cevichedbsync-operator/api/v1alpha1"
)

// PostgresSyncReconciler reconciles a PostgresSync object
type PostgresSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Constants for phases
const (
	PhasePending    = "Pending"
	PhaseInProgress = "InProgress"
	PhaseSucceeded  = "Succeeded"
	PhaseFailed     = "Failed"
)

// Reconcile handles PostgresSync resources
func (r *PostgresSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling PostgresSync", "namespacedName", req.NamespacedName)

	// Fetch the PostgresSync instance
	var pgSync cevichev1alpha1.PostgresSync
	if err := r.Get(ctx, req.NamespacedName, &pgSync); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch PostgresSync")
		return ctrl.Result{}, err
	}

	// Look up the StatefulSet the sync is attached to
	statefulSet := &appsv1.StatefulSet{}
	statefulSetKey := types.NamespacedName{
		Name:      pgSync.Spec.StatefulSetRef.Name,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, statefulSetKey, statefulSet); err != nil {
		if errors.IsNotFound(err) {
			// StatefulSet doesn't exist yet, requeue
			logger.Info("StatefulSet not found, requeueing", "statefulset", statefulSetKey)
			pgSync.Status.Phase = PhasePending
			pgSync.Status.Message = "Waiting for StatefulSet to be created"
			if err := r.Status().Update(ctx, &pgSync); err != nil {
				logger.Error(err, "unable to update PostgresSync status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		}
		logger.Error(err, "unable to fetch StatefulSet")
		return ctrl.Result{}, err
	}

	// Check if StatefulSet is ready
	if statefulSet.Status.ReadyReplicas == 0 {
		logger.Info("StatefulSet not ready, requeueing", "statefulset", statefulSetKey)
		pgSync.Status.Phase = PhasePending
		pgSync.Status.Message = "Waiting for StatefulSet to be ready"
		if err := r.Status().Update(ctx, &pgSync); err != nil {
			logger.Error(err, "unable to update PostgresSync status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// First time setup - check for existing dump
	if pgSync.Status.Phase != PhaseSucceeded {
		// Try to find and restore dump.sql if it exists
		restored, err := r.findAndRestoreDump(ctx, &pgSync)
		if err != nil {
			logger.Error(err, "Failed to restore dump")
			pgSync.Status.Phase = PhaseFailed
			pgSync.Status.Message = fmt.Sprintf("Failed to restore dump: %v", err)
			if updateErr := r.Status().Update(ctx, &pgSync); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return ctrl.Result{}, err
		}

		// Update status based on restore result
		if restored {
			pgSync.Status.Phase = PhaseSucceeded
			pgSync.Status.Message = "Database initialized from dump.sql"
		} else {
			pgSync.Status.Phase = PhaseSucceeded
			pgSync.Status.Message = "Ready - no existing dump found"
		}

		if err := r.Status().Update(ctx, &pgSync); err != nil {
			logger.Error(err, "unable to update PostgresSync status")
			return ctrl.Result{}, err
		}
	}

	// Handle dump on webhook if enabled
	if pgSync.Spec.DumpOnWebhook {
		logger.Info("DumpOnWebhook is true, creating database dump")

		// Create the dump
		if err := r.createDatabaseDump(ctx, &pgSync); err != nil {
			logger.Error(err, "failed to create database dump")
			pgSync.Status.Phase = PhaseFailed
			pgSync.Status.Message = fmt.Sprintf("Failed to dump database: %v", err)
			if updateErr := r.Status().Update(ctx, &pgSync); updateErr != nil {
				logger.Error(updateErr, "failed to update PostgresSync status")
			}
			return ctrl.Result{}, err
		}

		// Reset the DumpOnWebhook flag
		pgSync.Spec.DumpOnWebhook = false
		if err := r.Update(ctx, &pgSync); err != nil {
			logger.Error(err, "failed to update PostgresSync after dump")
			return ctrl.Result{}, err
		}

		// Update status
		pgSync.Status.Phase = PhaseSucceeded
		pgSync.Status.Message = "Database dump created successfully"
		pgSync.Status.LastSyncTime = metav1.Now()
		if err := r.Status().Update(ctx, &pgSync); err != nil {
			logger.Error(err, "unable to update PostgresSync status")
			return ctrl.Result{}, err
		}

		logger.Info("Database dump completed successfully")
	}

	return ctrl.Result{}, nil
}

// findAndRestoreDump looks for dump.sql in the git repository and restores it if found
func (r *PostgresSyncReconciler) findAndRestoreDump(ctx context.Context, pgSync *cevichev1alpha1.PostgresSync) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("Looking for existing dump.sql", "namespace", pgSync.Namespace, "name", pgSync.Name)

	// Get Git credentials
	gitSecret := &corev1.Secret{}
	gitSecretKey := types.NamespacedName{
		Name:      pgSync.Spec.GitCredentials.SecretName,
		Namespace: pgSync.Namespace,
	}
	if err := r.Get(ctx, gitSecretKey, gitSecret); err != nil {
		logger.Error(err, "unable to fetch Git credentials")
		return false, fmt.Errorf("failed to get Git credentials: %w", err)
	}
	gitUsername := string(gitSecret.Data["username"])
	gitPassword := string(gitSecret.Data["password"])

	// Clone repository
	repoDir, err := r.cloneRepository(pgSync.Spec.RepositoryURL, gitUsername, gitPassword)
	if err != nil {
		logger.Error(err, "failed to clone Git repository")
		return false, fmt.Errorf("failed to clone Git repository: %w", err)
	}

	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			logger.Error(err, "Failed to remove repo directory")
		}
	}()

	// Determine dump directory path based on GitOutputPath
	dumpDir := "dumps" // Default path
	if pgSync.Spec.DatabaseDumpPath != "" {
		dumpDir = pgSync.Spec.DatabaseDumpPath
	}

	// Create full path to dump directory
	dumpsDir := filepath.Join(repoDir, dumpDir)
	if _, err := os.Stat(dumpsDir); os.IsNotExist(err) {
		logger.Info("No dumps directory found, creating it", "path", dumpsDir)
		if err := os.MkdirAll(dumpsDir, 0755); err != nil {
			logger.Error(err, "failed to create dumps directory")
			return false, fmt.Errorf("failed to create dumps directory: %w", err)
		}
		return false, nil // No dumps to restore
	}

	// Check if dump.sql exists
	dumpFile := filepath.Join(dumpsDir, "dump.sql")
	if _, err := os.Stat(dumpFile); os.IsNotExist(err) {
		logger.Info("No dump.sql found")
		return false, nil // No dumps to restore
	}

	// Get database credentials
	dbSecret := &corev1.Secret{}
	dbSecretKey := types.NamespacedName{
		Name:      pgSync.Spec.DatabaseCredentials.SecretName,
		Namespace: pgSync.Namespace,
	}
	if err := r.Get(ctx, dbSecretKey, dbSecret); err != nil {
		return false, fmt.Errorf("failed to get database credentials: %w", err)
	}

	// Build connection parameters
	// Now using the service and service namespace from the CRD
	host := pgSync.Spec.DatabaseService.Name
	if host == "" {
		return false, fmt.Errorf("database service name is required")
	}

	// If service namespace is provided, use it for FQDN
	if pgSync.Spec.DatabaseService.Namespace != "" {
		host = fmt.Sprintf("%s.%s.svc.cluster.local", host, pgSync.Spec.DatabaseService.Namespace)
	}

	port := string(dbSecret.Data["port"])
	if port == "" {
		port = "5432" // Default PostgreSQL port
	}

	dbName := string(dbSecret.Data["database"])
	if dbName == "" {
		return false, fmt.Errorf("database name is required in secret")
	}

	dbUser := string(dbSecret.Data["username"])
	if dbUser == "" {
		return false, fmt.Errorf("database username is required in secret")
	}

	dbPassword := string(dbSecret.Data["password"])
	if dbPassword == "" {
		return false, fmt.Errorf("database password is required in secret")
	}

	// Execute psql to restore the database
	restoreCmd := exec.Command("psql",
		"-h", host,
		"-p", port,
		"-U", dbUser,
		"-d", dbName,
		"-f", dumpFile,
	)
	restoreCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbPassword))

	output, err := restoreCmd.CombinedOutput()
	if err != nil {
		logger.Error(err, "failed to restore database", "output", string(output))
		return false, fmt.Errorf("failed to restore database: %w, output: %s", err, output)
	}

	logger.Info("Database restore completed successfully")
	return true, nil
}

// createDatabaseDump creates a simple dump.sql and commits it to git
func (r *PostgresSyncReconciler) createDatabaseDump(ctx context.Context, pgSync *cevichev1alpha1.PostgresSync) error {
	logger := log.FromContext(ctx)
	logger.Info("Creating database dump", "namespace", pgSync.Namespace, "name", pgSync.Name)

	// Get database connection credentials
	dbSecret := &corev1.Secret{}
	dbSecretKey := types.NamespacedName{
		Name:      pgSync.Spec.DatabaseCredentials.SecretName,
		Namespace: pgSync.Namespace,
	}
	if err := r.Get(ctx, dbSecretKey, dbSecret); err != nil {
		logger.Error(err, "unable to fetch database credentials")
		return fmt.Errorf("failed to get database credentials: %w", err)
	}

	// Build connection parameters
	// Now using the service and service namespace from the CRD
	host := pgSync.Spec.DatabaseService.Name
	if host == "" {
		return fmt.Errorf("database service name is required")
	}

	// If service namespace is provided, use it for FQDN
	if pgSync.Spec.DatabaseService.Namespace != "" {
		host = fmt.Sprintf("%s.%s.svc.cluster.local", host, pgSync.Spec.DatabaseService.Namespace)
	}

	port := string(dbSecret.Data["port"])
	if port == "" {
		port = "5432" // Default PostgreSQL port
	}

	dbName := string(dbSecret.Data["database"])
	if dbName == "" {
		return fmt.Errorf("database name is required in secret")
	}

	dbUser := string(dbSecret.Data["username"])
	if dbUser == "" {
		return fmt.Errorf("database username is required in secret")
	}

	dbPassword := string(dbSecret.Data["password"])
	if dbPassword == "" {
		return fmt.Errorf("database password is required in secret")
	}

	// Get Git credentials
	gitSecret := &corev1.Secret{}
	gitSecretKey := types.NamespacedName{
		Name:      pgSync.Spec.GitCredentials.SecretName,
		Namespace: pgSync.Namespace,
	}
	if err := r.Get(ctx, gitSecretKey, gitSecret); err != nil {
		logger.Error(err, "unable to fetch Git credentials")
		return fmt.Errorf("failed to get Git credentials: %w", err)
	}
	gitUsername := string(gitSecret.Data["username"])
	gitPassword := string(gitSecret.Data["password"])

	// Clone repository
	repoDir, err := r.cloneRepository(pgSync.Spec.RepositoryURL, gitUsername, gitPassword)
	if err != nil {
		logger.Error(err, "failed to clone Git repository")
		return fmt.Errorf("failed to clone Git repository: %w", err)
	}

	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			logger.Error(err, "Failed to remove repo directory")
		}
	}()

	// Determine dump directory path based on GitOutputPath
	dumpDir := "dumps" // Default path
	if pgSync.Spec.DatabaseDumpPath != "" {
		dumpDir = pgSync.Spec.DatabaseDumpPath
	}

	// Create full path to dump directory
	dumpsDir := filepath.Join(repoDir, dumpDir)
	if err := os.MkdirAll(dumpsDir, 0755); err != nil {
		logger.Error(err, "failed to create dumps directory")
		return fmt.Errorf("failed to create dumps directory: %w", err)
	}

	// Create dump file named dump.sql
	dumpFilePath := filepath.Join(dumpsDir, "dump.sql")

	// Setup pg_dump command - now using the provided database user
	dumpCmd := exec.Command("pg_dump",
		"-h", host,
		"-p", port,
		"-U", dbUser,
		"-d", dbName,
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-privileges",
		"-f", dumpFilePath,
	)
	dumpCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbPassword))

	// Run dump
	if output, err := dumpCmd.CombinedOutput(); err != nil {
		logger.Error(err, "failed to create dump", "output", string(output))
		return fmt.Errorf("pg_dump failed: %w, output: %s", err, output)
	}

	// Commit and push changes
	commitMsg := "Updated database dump"
	if err := r.commitAndPushChanges(repoDir, gitUsername, gitPassword, commitMsg); err != nil {
		logger.Error(err, "failed to commit and push changes")
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	logger.Info("Successfully completed database dump")
	return nil
}

// cloneRepository clones the Git repository to a temporary directory
func (r *PostgresSyncReconciler) cloneRepository(repoURL, username, password string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "git-repo-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Clone the repository
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL: repoURL,
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})

	if err != nil {
		if errRemove := os.RemoveAll(tempDir); errRemove != nil {
			_ = errRemove
		}
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return tempDir, nil
}

// commitAndPushChanges commits and pushes changes to the Git repository
func (r *PostgresSyncReconciler) commitAndPushChanges(repoDir, username, password, commitMessage string) error {
	// Open the repository
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	if _, err := worktree.Add("."); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Ceviche DB Sync Operator",
			Email: "operator@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push changes
	err = repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cevichev1alpha1.PostgresSync{}).
		Owns(&appsv1.StatefulSet{}).
		// Add watch for StatefulSet events to handle scale up/down
		Watches(
			&appsv1.StatefulSet{},
			handler.EnqueueRequestsFromMapFunc(r.findPostgresSyncForStatefulSet),
		).
		Complete(r)
}

// findPostgresSyncForStatefulSet finds all PostgresSync resources that reference a StatefulSet
func (r *PostgresSyncReconciler) findPostgresSyncForStatefulSet(ctx context.Context, obj client.Object) []ctrl.Request {
	logger := log.FromContext(ctx)

	statefulset, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		logger.Error(nil, "Failed to convert object to StatefulSet", "object", obj)
		return nil
	}

	// Find all PostgresSync resources that reference this StatefulSet
	var syncList cevichev1alpha1.PostgresSyncList
	if err := r.List(ctx, &syncList, client.InNamespace(statefulset.Namespace)); err != nil {
		logger.Error(err, "Failed to list PostgresSync resources")
		return nil
	}

	// Create reconcile requests for PostgresSync resources referencing this StatefulSet
	requests := make([]ctrl.Request, 0)
	for _, sync := range syncList.Items {
		if sync.Spec.StatefulSetRef.Name == statefulset.Name {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      sync.Name,
					Namespace: sync.Namespace,
				},
			})
		}
	}

	return requests
}
