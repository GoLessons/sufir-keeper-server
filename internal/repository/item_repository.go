package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

type ItemRepository struct{ client *db.Client }

func NewItemRepository(client *db.Client) *ItemRepository { return &ItemRepository{client: client} }

func (r *ItemRepository) Create(ctx context.Context, rec model.ItemRecord) (model.ItemRecord, error) {
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = rec.CreatedAt
	var metaJSON []byte
	if rec.Meta != nil {
		metaJSON, _ = json.Marshal(rec.Meta)
	}
	ins := r.client.Builder.
		Insert("keep.items").
		Columns("id", "user_id", "title", "type", "data_encrypted", "data_nonce", "data_key_encrypted", "data_key_nonce", "kek_version", "meta", "created_at", "updated_at").
		Values(rec.ID, rec.UserID, rec.Title, rec.Type, rec.DataEncrypted, rec.DataNonce, rec.DataKeyEncrypted, rec.DataKeyNonce, rec.KEKVersion, sq.Expr("?::jsonb", metaJSON), rec.CreatedAt, rec.UpdatedAt)
	sqlStr, args, err := ins.ToSql()
	if err != nil {
		return model.ItemRecord{}, err
	}
	if _, err := r.client.SQL.ExecContext(ctx, sqlStr, args...); err != nil {
		return model.ItemRecord{}, err
	}
	if rec.File != nil {
		insFile := r.client.Builder.
			Insert("keep.items_files").
			Columns("item_id", "s3_bucket", "s3_key", "size", "sha256").
			Values(rec.ID, rec.File.S3Bucket, rec.File.S3Key, rec.File.Size, rec.File.SHA256)
		sqlStrFile, argsFile, err := insFile.ToSql()
		if err != nil {
			// Should probably delete the item here, but for simplicity we return error
			return model.ItemRecord{}, err
		}
		if _, err := r.client.SQL.ExecContext(ctx, sqlStrFile, argsFile...); err != nil {
			return model.ItemRecord{}, err
		}
	}
	return rec, nil
}

func (r *ItemRepository) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	q := r.client.Builder.Select("i.id", "i.user_id", "i.title", "i.type", "i.data_encrypted", "i.data_nonce", "i.data_key_encrypted", "i.data_key_nonce", "i.kek_version", "i.meta", "i.created_at", "i.updated_at", "f.s3_bucket", "f.s3_key", "f.size", "f.sha256").
		From("keep.items i").
		LeftJoin("keep.items_files f ON i.id = f.item_id").
		Where(sq.Eq{"i.id": id, "i.user_id": userID}).Limit(1)
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return model.ItemRecord{}, err
	}
	var rec model.ItemRecord
	var metaBytes []byte
	var s3Bucket, s3Key, sha256 *string
	var size *int64
	err = r.client.SQL.QueryRowContext(ctx, sqlStr, args...).Scan(&rec.ID, &rec.UserID, &rec.Title, &rec.Type, &rec.DataEncrypted, &rec.DataNonce, &rec.DataKeyEncrypted, &rec.DataKeyNonce, &rec.KEKVersion, &metaBytes, &rec.CreatedAt, &rec.UpdatedAt, &s3Bucket, &s3Key, &size, &sha256)
	if err != nil {
		return model.ItemRecord{}, err
	}
	if s3Bucket != nil && s3Key != nil && size != nil {
		rec.File = &model.ItemFile{
			S3Bucket: *s3Bucket,
			S3Key:    *s3Key,
			Size:     *size,
		}
		if sha256 != nil {
			rec.File.SHA256 = *sha256
		}
	}

	if len(metaBytes) > 0 {
		var meta map[string]string
		_ = json.Unmarshal(metaBytes, &meta)
		rec.Meta = meta
	}
	return rec, nil
}

type ListResult struct {
	Items []model.ItemListRecord
	Total int
}

func (r *ItemRepository) List(ctx context.Context, userID uuid.UUID, filterType *string, search *string, limit int, offset int) (ListResult, error) {
	b := r.client.Builder.Select("id", "title", "meta", "created_at", "updated_at").From("keep.items").Where(sq.Eq{"user_id": userID}).OrderBy("updated_at DESC", "created_at DESC").Limit(uint64(limit)).Offset(uint64(offset))
	if filterType != nil {
		b = b.Where(sq.Eq{"type": *filterType})
	}
	if search != nil && *search != "" {
		b = b.Where("title ILIKE ?", "%"+*search+"%")
	}
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return ListResult{}, err
	}
	rows, err := r.client.SQL.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	items := make([]model.ItemListRecord, 0, limit)
	for rows.Next() {
		var rec model.ItemListRecord
		var metaBytes []byte
		if err := rows.Scan(&rec.ID, &rec.Title, &metaBytes, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return ListResult{}, err
		}
		if len(metaBytes) > 0 {
			var meta map[string]string
			_ = json.Unmarshal(metaBytes, &meta)
			rec.Meta = meta
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}
	cntQ := r.client.Builder.Select("count(*)").From("keep.items").Where(sq.Eq{"user_id": userID})
	if filterType != nil {
		cntQ = cntQ.Where(sq.Eq{"type": *filterType})
	}
	if search != nil && *search != "" {
		cntQ = cntQ.Where("title ILIKE ?", "%"+*search+"%")
	}
	cntSQL, cntArgs, err := cntQ.ToSql()
	if err != nil {
		return ListResult{}, err
	}
	var total int
	if err := r.client.SQL.QueryRowContext(ctx, cntSQL, cntArgs...).Scan(&total); err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total}, nil
}

type ItemUpdateRecord struct {
	Title            *string
	Type             *string
	Meta             *map[string]string
	DataEncrypted    *[]byte
	DataNonce        *[]byte
	DataKeyEncrypted *[]byte
	DataKeyNonce     *[]byte
	KEKVersion       *int
}

func (r *ItemRepository) Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, upd ItemUpdateRecord) (model.ItemRecord, error) {
	b := r.client.Builder.Update("keep.items").Set("updated_at", sq.Expr("now()"))
	if upd.Title != nil {
		b = b.Set("title", *upd.Title)
	}
	if upd.Type != nil {
		b = b.Set("type", *upd.Type)
	}
	if upd.Meta != nil {
		if m, _ := json.Marshal(*upd.Meta); len(m) > 0 {
			b = b.Set("meta", sq.Expr("?::jsonb", m))
		} else {
			b = b.Set("meta", nil)
		}
	}
	if upd.DataEncrypted != nil {
		b = b.Set("data_encrypted", *upd.DataEncrypted)
	}
	if upd.DataNonce != nil {
		b = b.Set("data_nonce", *upd.DataNonce)
	}
	if upd.DataKeyEncrypted != nil {
		b = b.Set("data_key_encrypted", *upd.DataKeyEncrypted)
	}
	if upd.DataKeyNonce != nil {
		b = b.Set("data_key_nonce", *upd.DataKeyNonce)
	}
	if upd.KEKVersion != nil {
		b = b.Set("kek_version", *upd.KEKVersion)
	}
	b = b.Where(sq.Eq{"id": id, "user_id": userID})
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return model.ItemRecord{}, err
	}
	if _, err := r.client.SQL.ExecContext(ctx, sqlStr, args...); err != nil {
		return model.ItemRecord{}, err
	}
	return r.Get(ctx, userID, id)
}

func (r *ItemRepository) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	d := r.client.Builder.Delete("keep.items").Where(sq.Eq{"id": id, "user_id": userID})
	sqlStr, args, err := d.ToSql()
	if err != nil {
		return err
	}
	res, err := r.client.SQL.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ItemRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.client.SQL.BeginTx(ctx, &sql.TxOptions{})
}

func (r *ItemRepository) GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error) {
	q := r.client.Builder.Select("i.id", "i.user_id", "i.title", "i.type", "i.data_encrypted", "i.data_nonce", "i.data_key_encrypted", "i.data_key_nonce", "i.kek_version", "i.meta", "i.created_at", "i.updated_at", "f.s3_bucket", "f.s3_key", "f.size", "f.sha256").
		From("keep.items i").
		LeftJoin("keep.items_files f ON i.id = f.item_id").
		Where(sq.Eq{"i.id": id, "i.user_id": userID}).Limit(1)
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return model.ItemRecord{}, err
	}
	var rec model.ItemRecord
	var metaBytes []byte
	var s3Bucket, s3Key, sha256 *string
	var size *int64
	err = tx.QueryRowContext(ctx, sqlStr, args...).Scan(&rec.ID, &rec.UserID, &rec.Title, &rec.Type, &rec.DataEncrypted, &rec.DataNonce, &rec.DataKeyEncrypted, &rec.DataKeyNonce, &rec.KEKVersion, &metaBytes, &rec.CreatedAt, &rec.UpdatedAt, &s3Bucket, &s3Key, &size, &sha256)
	if err != nil {
		return model.ItemRecord{}, err
	}
	if s3Bucket != nil && s3Key != nil && size != nil {
		rec.File = &model.ItemFile{S3Bucket: *s3Bucket, S3Key: *s3Key, Size: *size}
		if sha256 != nil {
			rec.File.SHA256 = *sha256
		}
	}
	if len(metaBytes) > 0 {
		var meta map[string]string
		_ = json.Unmarshal(metaBytes, &meta)
		rec.Meta = meta
	}
	return rec, nil
}

func (r *ItemRepository) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error {
	d := r.client.Builder.Delete("keep.items").Where(sq.Eq{"id": id, "user_id": userID})
	sqlStr, args, err := d.ToSql()
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
