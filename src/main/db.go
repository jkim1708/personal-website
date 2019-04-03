package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

// request struct
type Request struct {
	RequestId   int
	Name        string
	CompanyName string
	EmailAdress string
}

func insertData(req *Request) error {
	db, err := sql.Open("mysql", "root:GrenzGraben1@tcp(127.0.0.1:3306)/personalWebsite_DB")

	if err != nil {
		fmt.Println("db1")
		panic(err.Error())
	}

	insert, err := db.Query(
		`INSERT INTO cvRequests (name,companyName,emailAdress) VALUES ('` + req.Name + `','` + req.CompanyName + `','` + req.EmailAdress + `')`)

	if err != nil {
		fmt.Println("db2")
		panic(err.Error())
	}

	defer insert.Close()
	defer db.Close()

	return nil
}

func fetchData() ([]*Request, error) {
	db, err := sql.Open("mysql", "root:GrenzGraben1@tcp(127.0.0.1:3306)/personalWebsite_DB")

	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	result, err := db.Query(`SELECT * FROM cvRequests`)

	if err != nil {
		panic(err.Error())
	}

	resultSet := make([]*Request, 0)

	for result.Next() {
		var request Request

		err = result.Scan(&request.RequestId, &request.Name, &request.CompanyName, &request.EmailAdress)

		if err != nil {
			panic(err.Error())
		}

		resultSet = append(resultSet, &request)

	}

	return resultSet, nil
}

func deleteData(name string) error {

	db, err := sql.Open("mysql", "root:GrenzGraben1@tcp(127.0.0.1:3306)/personalWebsite_DB")

	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	resultSet, err := searchData(name)

	if err != nil || len(resultSet) == 0 {
		return errors.New("db.deleteData(): cannot name data in DB")
	}

	result, err := db.Query(`DELETE FROM cvRequests WHERE request_id =` + strconv.Itoa(resultSet[0].RequestId))
	if err != nil {
		return errors.New("db.deleteData(): sql delete error")
	}
	defer result.Close()
	defer db.Close()

	return nil

}

func searchData(name string) ([]*Request, error) {
	db, err := sql.Open("mysql", "root:GrenzGraben1@tcp(127.0.0.1:3306)/personalWebsite_DB")

	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	result, err := db.Query(`SELECT * FROM cvRequests WHERE name=` + `'` + name + `'`)

	if err != nil {
		return nil, errors.New("db.serachData(): cannot find data in DB")
	}

	resultSet := make([]*Request, 0)

	for result.Next() {
		var request Request

		err = result.Scan(&request.RequestId, &request.Name, &request.CompanyName, &request.EmailAdress)

		if err != nil {
			return nil, errors.New("db.serachData(): cannot find data in DB")
		}

		resultSet = append(resultSet, &request)

	}

	return resultSet, nil
}
