package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cevichev1alpha1 "cevichedbsync-operator/api/v1alpha1"
)

// WebhookServer handles on-demand database dumps
type WebhookServer struct {
	Addr   string
	Client client.Client
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(addr string, client client.Client) *WebhookServer {
	return &WebhookServer{
		Addr:   addr,
		Client: client,
	}
}

// Start starts the webhook server
func (s *WebhookServer) Start() error {
	http.HandleFunc("/dump/", s.handleDumpRequest)
	return http.ListenAndServe(s.Addr, nil)
}

// handleDumpRequest handles database dump requests
// Path format: /dump/{namespace}/{name}
func (s *WebhookServer) handleDumpRequest(w http.ResponseWriter, r *http.Request) {
	logger := log.Log.WithName("webhook-server")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 || parts[1] != "dump" {
		http.Error(w, "Invalid path. Expected /dump/{namespace}/{name}", http.StatusBadRequest)
		return
	}

	namespace := parts[2]
	name := parts[3]

	// Trigger dump
	if err := s.triggerDatabaseDump(namespace, name); err != nil {
		logger.Error(err, "Failed to trigger database dump", "namespace", namespace, "name", name)
		http.Error(w, fmt.Sprintf("Failed to trigger database dump: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Database dump for %s/%s triggered", namespace, name),
	})
}

// triggerDatabaseDump triggers an on-demand database dump
func (s *WebhookServer) triggerDatabaseDump(namespace, name string) error {
	ctx := context.Background()

	// Get the PostgresSync resource
	pgSync := &cevichev1alpha1.PostgresSync{}
	if err := s.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, pgSync); err != nil {
		return fmt.Errorf("failed to get PostgresSync resource: %w", err)
	}

	// Set the DumpOnWebhook flag to trigger dump in reconciler
	pgSync.Spec.DumpOnWebhook = true
	if err := s.Client.Update(ctx, pgSync); err != nil {
		return fmt.Errorf("failed to update PostgresSync: %w", err)
	}

	return nil
}
