// Package main implements a simple shopping cart backend using Go Cloud's docstore.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	multicloud "github.com/udhos/golang-multicloud"
	"gocloud.dev/docstore"
	_ "gocloud.dev/docstore/awsdynamodb/v2"
	_ "gocloud.dev/docstore/gcpfirestore"
	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/docstore/mongodocstore"
	"gocloud.dev/gcerrors"
)

// Product is a read-only catalog item.
type Product struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// CartItem is one line in the shopping cart.
type CartItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

// Cart is the document stored in the docstore.
// The "ID" field is the partition/document key.
type Cart struct {
	ID    string     `json:"id"`
	Items []CartItem `json:"items"`
}

const cartDocID = "shopping-cart-1"

// catalog is the static list of available products.
var catalog = []Product{
	{ID: "p1", Name: "Go Programming Language Book", Price: 39.99},
	{ID: "p2", Name: "Cloud Architecture Guide", Price: 49.99},
	{ID: "p3", Name: "Kubernetes in Action", Price: 44.99},
	{ID: "p4", Name: "Microservices Patterns", Price: 34.99},
	{ID: "p5", Name: "Clean Code", Price: 29.99},
}

// Server holds the shared state for handler functions.
type Server struct {
	cloud       string
	docstoreURL string
	collection  *docstore.Collection
	mu          sync.Mutex // protects cart operations
}

func deriveCloud(docstoreURL string) string {
	scheme, _, _ := strings.Cut(docstoreURL, "://")
	switch scheme {
	case "dynamodb":
		return "aws"
	case "mongo":
		return "azure"
	case "firestore":
		return "gcp"
	case "mem":
		return "local"
	default:
		return scheme
	}
}

func main() {
	docstoreURL := os.Getenv("DOCSTORE_URL")
	if docstoreURL == "" {
		docstoreURL = "mem://carts/ID"
	}
	cloud := deriveCloud(docstoreURL)

	ctx := context.Background()
	coll, err := docstore.OpenCollection(ctx, docstoreURL)
	if err != nil {
		log.Fatalf("opening docstore %q: %v", docstoreURL, err)
	}
	defer coll.Close()

	srv := &Server{
		cloud:       cloud,
		docstoreURL: docstoreURL,
		collection:  coll,
	}

	sub, err := fs.Sub(multicloud.FrontendFiles, "frontend")
	if err != nil {
		log.Fatalf("embedding frontend: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/cloud", srv.handleCloud)
	mux.HandleFunc("/api/products", srv.handleProducts)
	mux.HandleFunc("/api/cart", srv.handleCart)
	mux.HandleFunc("/api/cart/add", srv.handleCartAdd)
	mux.HandleFunc("/api/cart/remove/", srv.handleCartRemove)
	mux.Handle("/", http.FileServer(http.FS(sub)))

	log.Printf("Shopping cart backend starting on :8080 (cloud=%s, docstore=%s)", cloud, docstoreURL)
	if err := http.ListenAndServe(":8080", corsMiddleware(mux)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// corsMiddleware adds permissive CORS headers for local development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// handleCloud returns the configured cloud provider name.
func (s *Server) handleCloud(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"cloud":        s.cloud,
		"docstore_url": s.docstoreURL,
	})
}

// handleProducts returns the static product catalog.
func (s *Server) handleProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

// handleCart returns the current cart.
func (s *Server) handleCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	s.mu.Lock()
	cart, err := s.getCart(ctx)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, fmt.Sprintf("getting cart: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// handleCartAdd adds (or increments) a product in the cart.
func (s *Server) handleCartAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ProductID string `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	// Look up product in catalog.
	var product *Product
	for i := range catalog {
		if catalog[i].ID == req.ProductID {
			product = &catalog[i]
			break
		}
	}
	if product == nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	s.mu.Lock()
	defer s.mu.Unlock()

	cart, err := s.getCart(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("getting cart: %v", err), http.StatusInternalServerError)
		return
	}
	// Find existing item or add new one.
	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == req.ProductID {
			cart.Items[i].Quantity++
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, CartItem{
			ProductID: product.ID,
			Name:      product.Name,
			Price:     product.Price,
			Quantity:  1,
		})
	}
	if err := s.saveCart(ctx, cart); err != nil {
		http.Error(w, fmt.Sprintf("saving cart: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// handleCartRemove removes a product from the cart entirely.
func (s *Server) handleCartRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract product ID from path: /api/cart/remove/{productId}
	productID := strings.TrimPrefix(r.URL.Path, "/api/cart/remove/")
	if productID == "" {
		http.Error(w, "product_id required in path", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	s.mu.Lock()
	defer s.mu.Unlock()

	cart, err := s.getCart(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("getting cart: %v", err), http.StatusInternalServerError)
		return
	}
	filtered := cart.Items[:0]
	for _, item := range cart.Items {
		if item.ProductID != productID {
			filtered = append(filtered, item)
		}
	}
	cart.Items = filtered
	if err := s.saveCart(ctx, cart); err != nil {
		http.Error(w, fmt.Sprintf("saving cart: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// getCart retrieves the cart document from the docstore.
// If the cart does not yet exist, an empty cart is returned (no error).
func (s *Server) getCart(ctx context.Context) (*Cart, error) {
	cart := &Cart{ID: cartDocID}
	if err := s.collection.Get(ctx, cart); err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return &Cart{ID: cartDocID, Items: []CartItem{}}, nil
		}
		return nil, err
	}
	if cart.Items == nil {
		cart.Items = []CartItem{}
	}
	return cart, nil
}

// saveCart writes (upserts) the cart document to the docstore.
func (s *Server) saveCart(ctx context.Context, cart *Cart) error {
	return s.collection.Put(ctx, cart)
}
