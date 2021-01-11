package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"net/http"

	"gorm.io/gorm"
	_ "gorm.io/driver/sqlite"
	"github.com/ekramul1z/newrelic-context"
	"github.com/ekramul1z/newrelic-context/nrgorm"
)

var db *gorm.DB

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./foo.db"),  &gorm.Config{})
	if err != nil {
		panic(err)
	}
	nrgorm.AddGormCallbacks(db)
	return db
}

type Product struct {
	ID   int
	Name string
}

func catalogPage(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var products []Product
		db := nrcontext.SetTxnToGorm(req.Context(), db)
		db.Find(&products)
		for i, v := range products {
			rw.Write([]byte(fmt.Sprintf("%v. %v\n", i, v.Name)))
		}
	})
}

func other_main() {
	db = initDB()

	handler := catalogPage(db)
	nrmiddleware, _ := nrcontext.NewMiddleware("test-app", "my-license-key")
	handler = nrmiddleware.Handler(handler)

	http.ListenAndServe(":8000", handler)
}
