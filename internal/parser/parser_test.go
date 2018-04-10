package parser

import (
	"testing"

	"github.com/aporeto-inc/manipulate"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_Parser(t *testing.T) {

	// Valid cases
	Convey("Given the filter: namespace = chris and test = true", t, func() {
		parser := NewFilterParser("namespace = chris and test = true")
		filter, err := parser.Parse()

		expectedFilter := manipulate.NewFilterComposer().And(
			manipulate.NewFilterComposer().WithKey("namespace").Equals("chris").Done(),
			manipulate.NewFilterComposer().WithKey("test").Equals(true).Done(),
		).Done()

		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	Convey("Given the filter: \"namespace\" = \"chris\" and \"test\" = true ", t, func() {
		parser := NewFilterParser("\"namespace\" = \"chris\" and \"test\" = true")
		filter, err := parser.Parse()

		expectedFilter := manipulate.NewFilterComposer().And(
			manipulate.NewFilterComposer().WithKey("namespace").Equals("chris").Done(),
			manipulate.NewFilterComposer().WithKey("test").Equals(true).Done(),
		).Done()

		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	Convey("Given the filter: \"age\" <= 32 or \"age\" > 50", t, func() {
		parser := NewFilterParser("\"age\" <= 32 or \"age\" > 50")
		filter, err := parser.Parse()

		expectedFilter := manipulate.NewFilterComposer().Or(
			manipulate.NewFilterComposer().WithKey("age").LesserThan(32).Done(),
			manipulate.NewFilterComposer().WithKey("age").GreaterThan(50).Done(),
		).Done()

		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	Convey("Given the filter: (\"file\" matches \"*.txt\" and \"file\" contains \"search\")", t, func() {
		parser := NewFilterParser("(\"file\" matches \"*.txt\" and \"file\" contains \"search\")")
		filter, err := parser.Parse()

		expectedFilter := manipulate.NewFilterComposer().And(
			manipulate.NewFilterComposer().WithKey("file").Matches("*.txt").Done(),
			manipulate.NewFilterComposer().WithKey("file").Contains("search").Done(),
		).Done()

		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	Convey("Given the filter: \"namespace\" = \"/chris\"", t, func() {
		parser := NewFilterParser("\"namespace\" = \"/chris\"")

		expectedFilter := manipulate.NewFilterComposer().And(
			manipulate.NewFilterComposer().WithKey("namespace").Equals("/chris").Done(),
		).Done()

		filter, err := parser.Parse()
		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	Convey("Given the filter: \"namespace\" = \"/chris\" and test = true and (\"name\" = toto or \"name\" = tata)", t, func() {
		parser := NewFilterParser("\"namespace\" = \"/chris\" and test = true (\"name\" = toto or \"name\" = tata)")

		expectedFilter := manipulate.NewFilterComposer().And(
			manipulate.NewFilterComposer().WithKey("namespace").Equals("/chris").Done(),
			manipulate.NewFilterComposer().WithKey("test").Equals(true).Done(),
			manipulate.NewFilterComposer().Or(
				manipulate.NewFilterComposer().WithKey("name").Equals("toto").Done(),
				manipulate.NewFilterComposer().WithKey("name").Equals("tata").Done(),
			).Done(),
		).Done()

		filter, err := parser.Parse()
		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	Convey("Given the filter: (name = toto or name = tata) and age == 31", t, func() {
		parser := NewFilterParser("(name = toto or name = tata) and age = 31")

		expectedFilter := manipulate.NewFilterComposer().Or(
			manipulate.NewFilterComposer().WithKey("name").Equals("toto").Done(),
			manipulate.NewFilterComposer().WithKey("name").Equals("tata").Done(),
		).Done().And(
			manipulate.NewFilterComposer().WithKey("age").Equals(31).Done(),
		).Done()

		filter, err := parser.Parse()
		So(err, ShouldEqual, nil)
		So(filter, ShouldNotEqual, nil)
		So(filter.String(), ShouldEqual, expectedFilter.String())
	})

	// Error cases
	Convey("Given the filter: namespace = chris and test = true or age = 31", t, func() {
		parser := NewFilterParser("namespace = chris and test = true or age = 31")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `misleading "or" condition. please add parentheses`)
	})

	Convey("Given the filter: \"namespace = chris", t, func() {
		parser := NewFilterParser("\"namespace = chris")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `missing quote after the word namespace`)
	})

	Convey("Given the filter: namespace\" = chris", t, func() {
		parser := NewFilterParser("namespace\" = chris")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `invalid operator. found "`)
	})

	Convey("Given the filter: namespace = \"chris", t, func() {
		parser := NewFilterParser("namespace = \"chris")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `missing quote after the value chris`)
	})

	Convey("Given the filter: namespace = chris\"", t, func() {
		parser := NewFilterParser("namespace = chris\"")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `missing quote before the value chris`)
	})

	Convey("Given the filter: namespace and chris\"", t, func() {
		parser := NewFilterParser("namespace and chris\"")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `invalid operator. found and`)
	})

	Convey("Given the filter: namespace = and\"", t, func() {
		parser := NewFilterParser("namespace = and\"")
		_, err := parser.Parse()

		So(err, ShouldNotEqual, nil)
		So(err.Error(), ShouldEqual, `invalid value. found and`)
	})

}

func Test_isLetter(t *testing.T) {
	Convey("Given I have a FilterParser", t, func() {
		So(isLetter('<'), ShouldEqual, true)
		So(isLetter('b'), ShouldEqual, true)
		So(isLetter(4), ShouldEqual, false)
	})
}
