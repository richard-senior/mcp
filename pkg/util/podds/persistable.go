package podds

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	_ "modernc.org/sqlite"
)

var db *sql.DB

// Persistable interface defines methods that persistent objects must implement
type Persistable interface {
	GetTableName() string
	GetPrimaryKey() map[string]interface{}
	SetPrimaryKey(map[string]interface{}) error
	BeforeSave() error
	AfterSave() error
	BeforeDelete() error
	AfterDelete() error
}

// CloseDatabase closes the database connection
func CloseDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// GetDB returns the database connection for testing purposes
func GetDB() (*sql.DB, error) {
	if db == nil {
		var err error
		db, err = sql.Open("sqlite", Config.PoddsDbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}

		// Test the connection
		if err = db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}

		logger.Info("Database initialized successfully", Config.PoddsDbPath)
	}
	return db, nil
}

// createTables creates all necessary database tables
func createTables() error {
	logger.Info("Creating database tables")

	// Create Match table
	if err := CreateTable(&Match{}); err != nil {
		return fmt.Errorf("failed to create match table: %w", err)
	}

	// Create Team table
	if err := CreateTable(&Team{}); err != nil {
		return fmt.Errorf("failed to create team table: %w", err)
	}

	// Create TeamStats table
	if err := CreateTable(&TeamStats{}); err != nil {
		return fmt.Errorf("failed to create team stats table: %w", err)
	}

	logger.Info("Database tables created successfully")
	return nil
}

// CreateTable creates a table for the given persistable object using struct tags
func CreateTable(obj Persistable) error {
	d, err := (GetDB())
	if err != nil {
		return err
	}

	tableName := obj.GetTableName()
	createSQL := generateCreateTableSQL(obj, tableName)

	logger.Debug("Creating table with SQL", createSQL)

	_, err = d.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	// Create indexes
	indexSQL := generateIndexSQL(obj, tableName)
	for _, query := range indexSQL {
		logger.Debug("Creating index with SQL", query)
		if _, err := d.Exec(query); err != nil {
			logger.Warn("Failed to create index", err)
		}
	}

	logger.Info("Table created successfully", tableName)
	return nil
}

// generateCreateTableSQL generates CREATE TABLE SQL from struct tags
func generateCreateTableSQL(obj interface{}, tableName string) string {
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	var columns []string
	var primaryKeys []string
	var foreignKeys []string

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields marked as non-persistable
		if field.Tag.Get("persist") == "false" || field.Tag.Get("db") == "-" {
			continue
		}

		// Get database column definition from tag
		dbType := field.Tag.Get("dbtype")
		if dbType == "" {
			continue // Skip fields without database type
		}

		// Get column name
		columnName := field.Tag.Get("column")
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		// Check if this is a primary key field
		if field.Tag.Get("primary") == "true" {
			primaryKeys = append(primaryKeys, columnName)
			// Remove PRIMARY KEY from individual column definition if it exists
			dbType = strings.ReplaceAll(dbType, "PRIMARY KEY", "")
			dbType = strings.TrimSpace(dbType)
		}

		columns = append(columns, fmt.Sprintf("%s %s", columnName, dbType))

		// Check for foreign key - format: "table.column"
		if fkRef := field.Tag.Get("fk"); fkRef != "" {
			fkParts := strings.Split(fkRef, ".")
			if len(fkParts) == 2 {
				referencedTable := fkParts[0]
				referencedColumn := fkParts[1]

				// Get foreign key action (default to RESTRICT)
				onDelete := field.Tag.Get("fk_delete")
				if onDelete == "" {
					onDelete = "RESTRICT"
				}

				onUpdate := field.Tag.Get("fk_update")
				if onUpdate == "" {
					onUpdate = "RESTRICT"
				}

				fkConstraint := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s ON UPDATE %s",
					columnName, referencedTable, referencedColumn, onDelete, onUpdate)
				foreignKeys = append(foreignKeys, fkConstraint)
			}
		}
	}

	// Add compound primary key constraint if we have multiple primary keys
	if len(primaryKeys) > 0 {
		pkConstraint := fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", "))
		columns = append(columns, pkConstraint)
	}

	// Add foreign key constraints
	for _, fk := range foreignKeys {
		columns = append(columns, fk)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ", "))
}

// generateIndexSQL generates index creation SQL from struct tags
func generateIndexSQL(obj interface{}, tableName string) []string {
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	var indexSQL []string

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		// Check for index tag
		indexTag := field.Tag.Get("index")
		if indexTag == "" {
			continue
		}

		columnName := field.Tag.Get("column")
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
		query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", indexName, tableName, columnName)
		indexSQL = append(indexSQL, query)
	}

	return indexSQL
}

// Save persists the object to the database (INSERT or UPDATE)
func Save(obj Persistable) error {
	// Call before save hook
	if err := obj.BeforeSave(); err != nil {
		return fmt.Errorf("before save hook failed: %w", err)
	}

	// Check if object exists
	exists, err := Exists(obj)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}

	if exists {
		err = update(obj)
	} else {
		err = insert(obj)
	}

	if err != nil {
		return err
	}

	// Call after save hook
	if err := obj.AfterSave(); err != nil {
		return fmt.Errorf("after save hook failed: %w", err)
	}

	return nil
}

// insert adds a new record to the database
func insert(obj Persistable) error {
	d, err := (GetDB())
	if err != nil {
		return err
	}

	tableName := obj.GetTableName()
	columns, placeholders, values := getInsertData(obj)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	logger.Debug("Insert SQL", query)

	_, err = d.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to insert into %s: %w", tableName, err)
	}

	return nil
}

// update modifies an existing record in the database
func update(obj Persistable) error {
	d, err := (GetDB())
	if err != nil {
		return err
	}

	tableName := obj.GetTableName()
	setPairs, values := getUpdateData(obj)

	whereClause, whereValues := buildWhereClause(obj.GetPrimaryKey())
	values = append(values, whereValues...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, strings.Join(setPairs, ", "), whereClause)

	logger.Debug("Update SQL", query)

	_, err = d.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to update %s: %w", tableName, err)
	}

	return nil
}

// getInsertData extracts column names, placeholders, and values for INSERT
func getInsertData(obj interface{}) ([]string, []string, []interface{}) {
	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)

	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
		objType = objType.Elem()
	}

	var columns []string
	var placeholders []string
	var values []interface{}

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields marked as non-persistable
		if field.Tag.Get("persist") == "false" || field.Tag.Get("db") == "-" {
			continue
		}

		// Skip fields without database type
		if field.Tag.Get("dbtype") == "" {
			continue
		}

		// Get column name
		columnName := field.Tag.Get("column")
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		columns = append(columns, columnName)
		placeholders = append(placeholders, "?")
		values = append(values, fieldValue.Interface())
	}

	return columns, placeholders, values
}

// getUpdateData extracts SET pairs and values for UPDATE
func getUpdateData(obj interface{}) ([]string, []interface{}) {
	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)

	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
		objType = objType.Elem()
	}

	var setPairs []string
	var values []interface{}

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields marked as non-persistable
		if field.Tag.Get("persist") == "false" || field.Tag.Get("db") == "-" {
			continue
		}

		// Skip fields without database type
		if field.Tag.Get("dbtype") == "" {
			continue
		}

		// Get column name
		columnName := field.Tag.Get("column")
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		// Skip primary key fields in updates
		if field.Tag.Get("primary") == "true" {
			continue
		}

		setPairs = append(setPairs, fmt.Sprintf("%s = ?", columnName))
		values = append(values, fieldValue.Interface())
	}

	return setPairs, values
}

// Exists checks if the object exists in the database
func Exists(obj Persistable) (bool, error) {
	d, err := (GetDB())
	if err != nil {
		return false, err
	}

	tableName := obj.GetTableName()
	whereClause, values := buildWhereClause(obj.GetPrimaryKey())

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", tableName, whereClause)

	var count int
	err = d.QueryRow(query, values...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existence in %s: %w", tableName, err)
	}

	return count > 0, nil
}

// Delete removes the object from the database
func Delete(obj Persistable) error {
	d, err := (GetDB())
	if err != nil {
		return err
	}

	// Call before delete hook
	if err := obj.BeforeDelete(); err != nil {
		return fmt.Errorf("before delete hook failed: %w", err)
	}

	tableName := obj.GetTableName()
	whereClause, values := buildWhereClause(obj.GetPrimaryKey())

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)

	_, err = d.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to delete from %s: %w", tableName, err)
	}

	// Call after delete hook
	if err := obj.AfterDelete(); err != nil {
		return fmt.Errorf("after delete hook failed: %w", err)
	}

	return nil
}

// FindByID retrieves an object by its ID
func FindByPrimaryKey(obj Persistable, primaryKey map[string]interface{}) error {
	d, err := (GetDB())
	if err != nil {
		return err
	}

	tableName := obj.GetTableName()
	columns, destinations := getSelectData(obj)
	whereClause, values := buildWhereClause(primaryKey)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(columns, ", "), tableName, whereClause)

	logger.Debug("FindByPrimaryKey SQL", query)

	row := d.QueryRow(query, values...)
	err = row.Scan(destinations...)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("record not found in %s", tableName)
		}
		return fmt.Errorf("failed to scan row from %s: %w", tableName, err)
	}

	return nil
}

// FindAll retrieves all records of the given type
func FindAll(obj Persistable) ([]interface{}, error) {
	d, err := (GetDB())
	if err != nil {
		return nil, err
	}

	tableName := obj.GetTableName()
	columns, _ := getSelectData(obj)

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)

	logger.Debug("FindAll SQL", query)

	rows, err := d.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", tableName, err)
	}
	defer rows.Close()

	var results []interface{}
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	for rows.Next() {
		// Create new instance of the object type
		newObj := reflect.New(objType).Interface()
		_, destinations := getSelectData(newObj)

		err := rows.Scan(destinations...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row from %s: %w", tableName, err)
		}

		results = append(results, newObj)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows from %s: %w", tableName, err)
	}

	return results, nil
}

// getSelectData extracts column names and scan destinations for SELECT
func getSelectData(obj interface{}) ([]string, []interface{}) {
	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)

	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
		objType = objType.Elem()
	}

	var columns []string
	var destinations []interface{}

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields without database type
		if field.Tag.Get("dbtype") == "" {
			continue
		}

		// Get column name
		columnName := field.Tag.Get("column")
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		columns = append(columns, columnName)
		destinations = append(destinations, fieldValue.Addr().Interface())
	}

	return columns, destinations
}

// BulkSave saves multiple objects in a transaction
func BulkSave(objects []Persistable) error {
	d, err := (GetDB())
	if err != nil {
		return err
	}

	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, obj := range objects {
		if err := Save(obj); err != nil {
			return fmt.Errorf("failed to save object: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// buildWhereClause builds a WHERE clause from a primary key map
func buildWhereClause(primaryKey map[string]interface{}) (string, []interface{}) {
	var conditions []string
	var values []interface{}

	for column, value := range primaryKey {
		conditions = append(conditions, fmt.Sprintf("%s = ?", column))
		values = append(values, value)
	}

	return strings.Join(conditions, " AND "), values
}

// getPrimaryKeyFields returns the primary key field names from struct tags
func getPrimaryKeyFields(obj interface{}) []string {
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	var primaryKeys []string

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check if this is a primary key field
		if field.Tag.Get("primary") == "true" {
			columnName := field.Tag.Get("column")
			if columnName == "" {
				columnName = strings.ToLower(field.Name)
			}
			primaryKeys = append(primaryKeys, columnName)
		}
	}

	return primaryKeys
}

// FindWhere executes a custom WHERE query
func FindWhere(obj Persistable, whereClause string, args ...interface{}) ([]interface{}, error) {
	d, err := (GetDB())
	if err != nil {
		return nil, err
	}

	tableName := obj.GetTableName()
	columns, _ := getSelectData(obj)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(columns, ", "), tableName, whereClause)

	logger.Debug("FindWhere SQL", query)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", tableName, err)
	}
	defer rows.Close()

	var results []interface{}
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	for rows.Next() {
		// Create new instance of the object type
		newObj := reflect.New(objType).Interface()
		_, destinations := getSelectData(newObj)

		err := rows.Scan(destinations...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row from %s: %w", tableName, err)
		}

		results = append(results, newObj)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows from %s: %w", tableName, err)
	}

	return results, nil
}
