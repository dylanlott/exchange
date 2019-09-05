package db

import "context"

// WithTx gives you a callback function that exposes any errors that occur
func (db *DB) WithTx(ctx context.Context,
	fn func(context.Context, *Tx) error) (err error) {

	tx, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback() // log this perhaps?
		}
	}()
	return fn(ctx, tx)
}
