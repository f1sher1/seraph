package objects

import (
	"context"
	"fmt"
	"seraph/app/db/models"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"time"

	"github.com/google/uuid"
)

type NamedLock struct {
	*models.NamedLock
	ContextObject
	PersistentObject
}

func (l *NamedLock) Save(ctx *contextx.Context) error {
	if !l.IsCreated() {
		l.CreatedAt = time.Now().UTC()
		if l.ID == "" {
			l.ID = uuid.NewString()
		}
		l.UpdatedAt = l.CreatedAt
	} else {
		l.UpdatedAt = time.Now().UTC()
	}

	lockModel := l.GetQuery(ctx).NamedLock

	err := lockModel.WithContext(context.Background()).Save(l.NamedLock)
	if err != nil {
		return err
	}
	l.SetContext(ctx)
	l.SetCreated()
	return nil
}

func (l *NamedLock) Delete(ctx *contextx.Context) error {
	if !l.IsCreated() {
		return fmt.Errorf("object %s isn't a persistent object, can't delete it", l.ID)
	}
	lockModel := l.GetQuery(ctx).NamedLock
	_, err := lockModel.WithContext(context.Background()).Where(lockModel.ID.Eq(l.ID)).Delete()
	return err
}

func NewNamedLock() *NamedLock {
	return &NamedLock{NamedLock: &models.NamedLock{}}
}

func WithNamedLock(ctx *contextx.Context, name string, callback func() error) error {
	locker := NewNamedLock()
	locker.Name = name
	err := locker.Save(ctx)
	if err != nil {
		return err
	}

	err = callback()
	delErr := locker.Delete(ctx)
	if delErr != nil {
		log.Warnf(ctx, "clear lock %s failed, error: %s", name, delErr.Error())
	}
	return err
}
