package models

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/Abhishek-Mali-Simform/assessments/database"
)

type PersonInfo struct {
	Name        string `json:"name"`
	Age         int    `json:"age"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

func RetrievePerson(personID int) (*PersonInfo, error) {
	if personID <= 0 {
		return nil, errors.New("no personID passed to retrieve person")
	}
	personInfo := new(PersonInfo)
	query := `
		SELECT 
            p.name, p.age, ph.number, a.city, a.state, a.street1, a.street2, a.zip_code
        FROM 
            person p
        JOIN 
            phone ph ON p.id = ph.person_id
        JOIN 
            address_join aj ON p.id = aj.person_id
        JOIN 
            address a ON aj.address_id = a.id
        WHERE 
            p.id =` + strconv.Itoa(personID)
	err := database.DB.QueryRow(query).Scan(&personInfo.Name, &personInfo.Age, &personInfo.PhoneNumber, &personInfo.City, &personInfo.State, &personInfo.Street1, &personInfo.Street2, &personInfo.ZipCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("person not found")
		}
		return nil, err
	}
	return personInfo, nil
}

func (personInfo *PersonInfo) Save() error {
	tx, err := database.DB.Begin()
	if err != nil {
		return errors.New("failed to begin transaction: " + err.Error())
	}

	var personID int
	if database.DriverName == "mysql" {
		res, err := tx.Exec("INSERT INTO person (name, age) VALUES (?, 0)", personInfo.Name)
		if err == nil {
			personID64, errID := res.LastInsertId()
			if errID != nil {
				rollBackErr := tx.Rollback()
				if rollBackErr != nil {
					return errors.New("failed to rollback transaction: " + errID.Error())
				}
			}
			personID = int(personID64)
		}
	} else {
		err = tx.QueryRow("INSERT INTO person (name, age) VALUES ($1, $2) RETURNING id", personInfo.Name, personInfo.Age).Scan(&personID)
	}
	if err != nil {
		rollBackErr := tx.Rollback()
		if rollBackErr != nil {
			return errors.New("failed to rollback transaction: " + err.Error())
		}
		return errors.New("failed to insert person: " + err.Error())
	}
	phoneQuery := "INSERT INTO phone (number, person_id) VALUES "
	if database.DriverName == "mysql" {
		phoneQuery += "(?, ?)"
	} else {
		phoneQuery += "($1, $2)"
	}
	_, err = tx.Exec(phoneQuery, personInfo.PhoneNumber, personID)
	if err != nil {
		rollBackErr := tx.Rollback()
		if rollBackErr != nil {
			return errors.New("failed to rollback transaction: " + err.Error())
		}
		return errors.New("failed to insert phone number: " + err.Error())
	}

	var addressID int
	if database.DriverName == "mysql" {
		res, err := tx.Exec("INSERT INTO address (city, state, street1, street2, zip_code) VALUES (?, ?, ?, ?, ?)",
			personInfo.City, personInfo.State, personInfo.Street1, personInfo.Street2, personInfo.ZipCode)
		if err == nil {
			addressID64, errID := res.LastInsertId()
			if errID != nil {
				rollBackErr := tx.Rollback()
				if rollBackErr != nil {
					return errors.New("failed to rollback transaction: " + errID.Error())
				}
			}
			addressID = int(addressID64)
		}
	} else {
		err = tx.QueryRow("INSERT INTO address (city, state, street1, street2, zip_code) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			personInfo.City, personInfo.State, personInfo.Street1, personInfo.Street2, personInfo.ZipCode).Scan(&addressID)
	}
	if err != nil {
		rollBackErr := tx.Rollback()
		if rollBackErr != nil {
			return errors.New("failed to rollback transaction: " + err.Error())
		}
		return errors.New("failed to insert address: " + err.Error())
	}

	addressJoinQuery := "INSERT INTO address_join (person_id, address_id) VALUES "
	if database.DriverName == "mysql" {
		addressJoinQuery += "(?, ?)"
	} else {
		addressJoinQuery += "($1, $2)"
	}

	_, err = tx.Exec(addressJoinQuery, personID, addressID)
	if err != nil {
		rollBackErr := tx.Rollback()
		if rollBackErr != nil {
			return errors.New("failed to rollback transaction: " + err.Error())
		}
		return errors.New("failed to insert address_join: " + err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return errors.New("failed to commit person info")
	}
	return nil
}
