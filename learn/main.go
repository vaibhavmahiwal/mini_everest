// package main

// import (
// 	"fmt"
// )

// type counter struct {
// 	Value int
// }

// func Increase_value(c counter) {
// 	c.Value++
// }
// func Increase_pointer(c *counter) {
// 	c.Value++
// }
// func main() {
// 	c := &counter{Value: 0}
// 	fmt.Println("counter value before increase: ", c.Value)
// 	Increase_value(*c)
// 	fmt.Println("counter value after increase: ", c.Value)
// 	Increase_pointer(c)
// 	fmt.Println("counter value after pointer increase: ", c.Value)
// }

// an in memory reconciler that makes an actual map
// match a desired state map
// this is a simple example of how controler work in kunernetes
package main

import "fmt"

//this is a database struct that has a name and a number of replicas
type Database struct {
	Name     string
	Replicas int
}

//this is a store interface that has methods to create or update,delete and reconcile the database
type Store interface {
	CreateOrUpdate(d Database) bool
	Delete(name string)
}

//this is an in memory store that implements the store interface
type InMemoryStore struct {
	data map[string]*Database
}

//this is a constructor for the in memory store
func NewStore() *InMemoryStore {
	//this is a map that hold the database name as key
	//and the database struct as value
	//make is a buitin fun that creats a map with the given type and size
	return &InMemoryStore{data: make(map[string]*Database)}
}

//this is a methhod that creates or updatesa database in the store
func (s *InMemoryStore) CreateOrUpdate(d Database) bool {
	//check if the database already exists in the store
	if actual, ok := s.data[d.Name]; ok {
		//if it exists check if the number of replicas is different
		if actual.Replicas != d.Replicas {
			//if diff then update the number of replicas and return values
			fmt.Printf("update %s %d->%d\n", d.Name, actual.Replicas, d.Replicas)
			actual.Replicas = d.Replicas
			return true
		}
		fmt.Printf("no-op %s\n", d.Name)
		return false
	}
	//if does not exist then creates a new database in the store and return the
	//true value
	s.data[d.Name] = &Database{Name: d.Name, Replicas: d.Replicas}
	fmt.Printf("create %s replicas=%d\n", d.Name, d.Replicas)
	return true
}

//this function is a method that delete a database from the store
func (s *InMemoryStore) Delete(name string) {
	if _, ok := s.data[name]; ok {
		delete(s.data, name)
		fmt.Printf("delete %s\n", name)
	}
}

//this function is a method that reconciles the desired state of the store with the actual state
// we range over the sesred state and call the create or update method for each database
func (s *InMemoryStore) Reconcile(desired map[string]Database) {
	for _, d := range desired {
		s.CreateOrUpdate(d)
	}
	//if the database is not in the desired state then we delete it from the store
	for name := range s.data {
		if _, ok := desired[name]; !ok {
			s.Delete(name)
		}
	}
}

//finally in this main function we create a new store and call the reconcile method
//with diff desired states to see how the store changes
func main() {
	s := NewStore()
	desired := map[string]Database{"db-a": {Name: "db-a", Replicas: 1}}
	fmt.Println("--- run 1 ---")
	s.Reconcile(desired)
	fmt.Println("--- run 2 (same) ---")
	s.Reconcile(desired)
	fmt.Println("--- run 3 (update) ---")
	desired["db-a"] = Database{Name: "db-a", Replicas: 3}
	s.Reconcile(desired)
	fmt.Println("--- run 4 (delete) ---")
	s.Reconcile(map[string]Database{})
}
