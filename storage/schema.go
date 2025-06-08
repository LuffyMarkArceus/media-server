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
	Size      int64     `json:"size"` // Using int64 for size as it's an 'int' in TS and can be large.
	URL       string    `json:"url"`
	ParentID  int64     `json:"parentId"` // References Folder.ID
	CreatedAt time.Time `json:"createdAt"`
}

// Folder represents a row in the folders_table.
type Folder struct {
	ID        int64         `json:"id"`
	OwnerID   string        `json:"ownerId"`
	Name      string        `json:"name"`
	ParentID  sql.NullInt64 `json:"parentId"` // Use sql.NullInt64 for nullable parent ID (references Folder.ID for nested folders)
	CreatedAt time.Time     `json:"createdAt"`
}

const CreateFilesTableSQL = `
CREATE TABLE IF NOT EXISTS files_table (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ownerId TEXT NOT NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    url TEXT NOT NULL,
    parent INTEGER NOT NULL, -- Foreign key to folders_table
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    CONSTRAINT fk_parent
        FOREIGN KEY (parent)
        REFERENCES folders_table(id)
        ON DELETE CASCADE -- Optional: Define behavior on parent folder deletion
);
`

const CreateFoldersTableSQL = `
CREATE TABLE IF NOT EXISTS folders_table (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ownerId TEXT NOT NULL,
    name TEXT NOT NULL,
    parent INTEGER, -- Nullable: Foreign key to self for nested folders
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Indexes (these can be created separately or within the table definition)
    CONSTRAINT fk_parent_folder
        FOREIGN KEY (parent)
        REFERENCES folders_table(id)
        ON DELETE CASCADE -- Optional: Define behavior on parent folder deletion
);
`
// You can also add separate CREATE INDEX statements for clarity or if not using constraints directly
const CreateFilesParentIndexSQL = `CREATE INDEX IF NOT EXISTS files_parent_index ON files_table (parent);`
const CreateFilesOwnerIDIndexSQL = `CREATE INDEX IF NOT EXISTS files_ownerId_index ON files_table (ownerId);`
const CreateFoldersParentIndexSQL = `CREATE INDEX IF NOT EXISTS folders_parent_index ON folders_table (parent);`
const CreateFoldersOwnerIDIndexSQL = `CREATE INDEX IF NOT EXISTS folders_ownerId_index ON folders_table (ownerId);`
