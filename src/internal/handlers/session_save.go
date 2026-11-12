package handlers

import (
	"context"

	"github.com/gofiber/fiber/v3/middleware/session"
)

// sessionByID is the part of *session.Store that saveLiveSession needs.
type sessionByID interface {
	GetByID(ctx context.Context, id string) (*session.Session, error)
}

// saveLiveSession saves sess and then confirms that the store still holds it.
//
// From Fiber v3.7.0, Save returns nil without writing when another request in
// this process has destroyed the session since it was loaded. A request that
// was in flight during a logout would otherwise continue as authenticated, so
// the caller must treat an error here as a failed save.
func saveLiveSession(ctx context.Context, store sessionByID, sess *session.Session) error {
	if err := sess.Save(); err != nil {
		return err
	}
	current, err := store.GetByID(ctx, sess.ID())
	if err != nil {
		return err
	}
	current.Release()
	return nil
}
