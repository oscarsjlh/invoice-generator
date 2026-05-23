package handler

import (
	"fmt"
	"net/http"

	"invoice-app/internal/db"
)

type StoreProvider interface {
	ForRequest(r *http.Request, user *db.User) (db.Storer, *db.User, error)
}

type RequestStoreProvider struct {
	multiStore  *db.MultiStore
	legacyStore *db.Store
}

func NewRequestStoreProvider(multiStore *db.MultiStore) *RequestStoreProvider {
	return &RequestStoreProvider{multiStore: multiStore}
}

func (p *RequestStoreProvider) SetLegacyStore(store *db.Store) {
	p.legacyStore = store
}

func (p *RequestStoreProvider) ForRequest(r *http.Request, user *db.User) (db.Storer, *db.User, error) {
	if p.legacyStore != nil {
		return p.legacyStore, &db.User{ID: 0, Username: "anonymous", DisplayName: "Anonymous"}, nil
	}
	if user == nil {
		return nil, nil, fmt.Errorf("authenticated user is required")
	}
	if p.multiStore == nil {
		return nil, nil, fmt.Errorf("store provider is not configured")
	}
	store, err := p.multiStore.ForUser(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return store, user, nil
}

func (p *RequestStoreProvider) ForUser(userID int64) (db.Storer, error) {
	if p.legacyStore != nil {
		return p.legacyStore, nil
	}
	if p.multiStore == nil {
		return nil, fmt.Errorf("store provider is not configured")
	}
	return p.multiStore.ForUser(userID)
}
