package storage

import (
	"database/sql"
	"time"
)

// File represents a row in the files_table.
type File struct {
	ID        int64     `json:"id"`
	OwnerID   string    `json:"ownerId"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	URL       string    `json:"url"`
	Type      string    `json:"type"`
	ParentID  int64     `json:"parentId"`
	CreatedAt time.Time `json:"createdAt"`
    ThumbnailURL *string   `json:"thumbnailUrl,omitempty"`
	SubtitleURL  *string   `json:"subtitleUrl,omitempty"`
}

// Folder represents a row in the folders_table.
type Folder struct {
	ID        int64         `json:"id"`
	OwnerID   string        `json:"ownerId"`
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	ParentID  sql.NullInt64 `json:"parentId"`
	CreatedAt time.Time     `json:"createdAt"`
}

const CreateFoldersTableSQL = `
CREATE TABLE IF NOT EXISTS folders_table (
    id SERIAL PRIMARY KEY,
    ownerId TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    parent INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    subtitle_gen_failed BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_parent_folder
        FOREIGN KEY (parent)
        REFERENCES folders_table(id)
        ON DELETE CASCADE
);
`

const CreateFilesTableSQL = `
CREATE TABLE IF NOT EXISTS files_table (
    id SERIAL PRIMARY KEY,
    ownerId TEXT NOT NULL,
    name TEXT NOT NULL,
    size BIGINT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    parent INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    thumbnail_url TEXT,
    subtitle_url TEXT,
    CONSTRAINT fk_parent
        FOREIGN KEY (parent)
        REFERENCES folders_table(id)
        ON DELETE CASCADE
);
`

const CreateFilesParentIndexSQL = `CREATE INDEX IF NOT EXISTS files_parent_index ON files_table (parent);`
const CreateFilesOwnerIDIndexSQL = `CREATE INDEX IF NOT EXISTS files_ownerId_index ON files_table (ownerId);`
const CreateFoldersParentIndexSQL = `CREATE INDEX IF NOT EXISTS folders_parent_index ON folders_table (parent);`
const CreateFoldersOwnerIDIndexSQL = `CREATE INDEX IF NOT EXISTS folders_ownerId_index ON folders_table (ownerId);`
