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

//an in memory reconciler that makes an actual map
//match a desired state map

package main

type database struct {
	Name    string
	Replica int
}
