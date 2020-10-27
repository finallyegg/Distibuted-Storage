package store

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db  *sql.DB
	err error
)

func CreateDB(path string) {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}
	db.Exec("CREATE TABLE IF NOT EXISTS chunks (sig CHAR(70) PRIMARY KEY, data BLOB)") // allow custom types

	rows, err := db.Query("SELECT sig, LENGTH(data) FROM chunks")
	if err != nil {
		PrintExit("SELECT DB error: %v\n", err)
	}
	defer rows.Close()

	sigs := 0
	totalSz := 0

	for rows.Next() {
		var sig string
		var sz int
		rows.Scan(&sig, &sz)
		sigs++
		totalSz += sz
		// PrintDebug("db chunk %q, sz %d\n", sig, sz)
	}
	PrintAlways("db %q has %d chunks, %d bytes\n", path, sigs, totalSz)
}

func InsertIntoDB(path string, sig string, data []byte) {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	// defer tx.Rollback()
	stmt, err := tx.Prepare("INSERT OR IGNORE INTO chunks(sig,data) VALUES(?,?)")
	if err != nil {
		PrintExit("INSERT DB error: %v\n", err)
	}

	_, err = stmt.Exec(sig, data)
	if err != nil {
		PrintExit("INSERT DB error: %v\n", err)
	}

	err = tx.Commit()
	if err != nil {
		PrintExit("insert - COMMIT DB error: %v\n", err)
	}
	stmt.Close() //runs here!

}

func updateFileDB(path string, sig string, data []byte) {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	// defer tx.Rollback()
	stmt, err := tx.Prepare("UPDATE chunks SET data = ? WHERE sig = ?")
	if err != nil {
		PrintExit("UPDATE DB error: %v\n", err)
	}

	_, err = stmt.Exec(data, sig)
	if err != nil {
		PrintExit("UPDATE DB error: %v\n", err)
	}

	err = tx.Commit()
	if err != nil {
		PrintExit("UPDATE - COMMIT DB error: %v\n", err)
	}
	stmt.Close() //runs here!
}

func DelFromDB(path string, sig string) error {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	var sql string
	if sig == "all" {
		sql = "DELETE FROM chunks"
		stmt, err := tx.Prepare(sql)
		if err != nil {
			PrintExit("Delete DB error: %v\n", err)
		}

		_, err = stmt.Exec()
		defer stmt.Close()
	} else {
		sql = "DELETE FROM chunks WHERE sig = ?"
		stmt, err := tx.Prepare(sql)
		if err != nil {
			PrintExit("Delete DB error: %v\n", err)
		}

		_, err = stmt.Exec(sig)
		defer stmt.Close()

	}

	if err != nil {
		PrintExit("Delete DB error: %v\n", err)
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println(sig)
		PrintExit("del - COMMIT DB error: %v\n", err)

	}

	return err
}

func GetAllFromDB(path string) []string {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT sig, data FROM chunks")
	if err != nil {
		PrintExit("SELECT DB error: %v\n", err)
	}
	defer rows.Close()

	var retVal []string
	for rows.Next() {
		var sig string
		var data []byte
		err = rows.Scan(&sig, &data)
		retVal = append(retVal, sig)
		if err != nil {
			PrintExit("SCAN DB result error: %v\n", err)
		}
	}
	return retVal
}

func GetFromDB(path string, sig string) ([]byte, error) {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}

	rows, err := db.Query("SELECT sig, data FROM chunks WHERE sig = ? ", sig)
	if err != nil {
		PrintExit("SELECT DB error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sig string
		var data []byte
		err = rows.Scan(&sig, &data)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, nil
}

func InfoDB(path string) (string, error) {
	if db, err = sql.Open("sqlite3", path); err != nil {
		PrintExit("Open DB %q error: %v\n", path, err)
	}

	rows, err := db.Query("SELECT sig, LENGTH(data) FROM chunks")
	if err != nil {
		PrintExit("INFO DB error: %v\n", err)
		return "", err
	}

	defer rows.Close()

	sigs := 0
	totalSz := 0

	for rows.Next() {
		var sig string
		var sz int
		rows.Scan(&sig, &sz)
		sigs++
		totalSz += sz
	}

	retVal := fmt.Sprintf("%d chunks, %d total bytes", sigs, totalSz)
	// println(retVal)
	return retVal, err
}
