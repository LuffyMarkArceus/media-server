package storage

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)


func InitDB(dataSourceName string) (* sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	if err = db.Ping(); err != nil{
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = db.Exec(CreateFoldersTableSQL)
	if err!= nil {
		return nil, fmt.Errorf("faied to create folders_table: %w", err)
	}
	log.Println("Created/Already Exists Table: folders_table")

	_, err = db.Exec(CreateFilesTableSQL)
	if err!= nil {
		return nil, fmt.Errorf("failed to create files_table: %w", err)
	}
	log.Println("Created/Already Exists Table: files_table")


	// Create additional indexes if not already part of table definition (though SQLite often optimizes FKs)
	// _, err = db.Exec(CreateFilesParentIndexSQL)
	// if err != nil {
	// 	log.Printf("Warning: failed to create files_parent_index: %v", err)
	// }
	// _, err = db.Exec(CreateFilesOwnerIDIndexSQL)
	// if err != nil {
	// 	log.Printf("Warning: failed to create files_ownerId_index: %v", err)
	// }
	// _, err = db.Exec(CreateFoldersParentIndexSQL)
	// if err != nil {
	// 	log.Printf("Warning: failed to create folders_parent_index: %v", err)
	// }
	// _, err = db.Exec(CreateFoldersOwnerIDIndexSQL)
	// if err != nil {
	// 	log.Printf("Warning: failed to create folders_ownerId_index: %v", err)
	// }

	return db, nil
}

