// Package graphtest is the test toolkit for a program built on the graph
// package. It holds a program's struct of node and group handles to the
// graph those handles were defined on, so a test catches a field whose
// handle is missing or names another node, and nothing in the graph package
// exists for the tests' sake.
//
// The package exports:
//
//   - [CheckNames], a test's check that each handle field of a struct of
//     node and group handles holds a defined handle whose name, lowercased
//     with its spaces removed, is the field's name lowercased; it reports
//     each failure as a test error without stopping the test
//   - [NameErrors], the errors CheckNames reports, for a caller that
//     handles them itself
package graphtest
