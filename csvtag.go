package csvtag

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

type fn func(*os.File) *string

// Config struct to pass to the Load function
type Config struct {
	Path      string
	Dest      interface{}
	Separator rune
	Header    []string
	Modifier  fn
}

// Load - Load a csv file and put it in a array of the dest type
// This uses tags
// Example:
// 	tabT := []Test{}
// 	err  := Load(Config{
// 			path: "csv_files/valid.csv",
// 			dest: &tabT,
// 			separator: ';',
// 			header: []string{"header1", "header2", "header3"}
// 		})
// The 'separator' and 'header' properties of the config object are optionals
// @param dest: object where to store the result
// @return an error if one occure
func Load(config Config) error {
	header, content, err := readFile(config.Path, config.Separator, config.Header, config.Modifier)
	if err != nil {
		return fmt.Errorf("Error while loading csv (%v):\n	==> %v", config.Path, err)
	}
	// This means that the file is empty
	if content == nil {
		return nil
	}
	// If there is some header in the config, don't skip the first line
	start := 1
	if config.Header != nil {
		start = 0
	}
	// Map content to the destination
	err = mapToDest(header, content[start:], config.Dest)
	if err != nil {
		return fmt.Errorf("Error while mapping the content to the destination:\n	==> %v", err)
	}
	return nil
}

// Load the header and file content in memory
// @param path: path of the csv file
// @param separator: the separator used in the csv file
// @param header: the optional header if the file does not contain one
func readFile(path string, separator rune, header []string) (map[string]int, [][]string, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	// Calling file mofifier
	var contents *string
	// if fn != nil {
	contents = modifier(file)
	// }
	// Read the file
	// We need to read it all at once to have the number of records
	var reader *csv.Reader
	if contents == nil {
		reader = csv.NewReader(file) // Create the csv reader from os.Open
	} else {
		reader = csv.NewReader(strings.NewReader(*contents)) // Create the csv reader from string
	}
	reader.TrimLeadingSpace = true
	if separator != 0 {
		reader.Comma = separator
	}
	content, err := reader.ReadAll() // Read all the file and put it in content
	file.Close()                     // Close the file
	if err != nil {
		return nil, nil, err
	}
	// If file is empty, return
	if len(content) == 0 {
		return nil, nil, nil
	}
	// Map each header name to its index
	// This will be used in the mapping
	// If there is no header in the config, treate first line as the header
	rawHeader := header
	if rawHeader == nil {
		rawHeader = content[0]
	}
	headerMap := make(map[string]int) // Create map
	for i, name := range rawHeader {
		headerMap[name] = i
	}
	// Return our header and content
	return headerMap, content, nil
}

// Map the provided content to the dest using the header and the tags
// @param header: the csv header to match with the struct's tags
// @param content: the content to put in dest
// @param dest: the destination where to put the file's content
func mapToDest(header map[string]int, content [][]string, dest interface{}) error {
	// Check destination is not nit
	if dest == nil {
		return fmt.Errorf("Destination slice is nil")
	}
	// Check destination is a slice
	if reflect.TypeOf(dest).Elem().Kind() != reflect.Slice {
		return fmt.Errorf("Destination is not a slice")
	}
	// Create the slice the put the values in
	// Get the reflected value of dest
	destRv := reflect.ValueOf(dest).Elem()
	// Create a new reflected value containing a slice:
	//   type    : dest's type
	//   length  : content's length
	//   capacity: content's length
	sliceRv := reflect.MakeSlice(destRv.Type(), len(content), len(content))
	// Map the records into the created slice
	for i, record := range content {
		item := sliceRv.Index(i) // Get the ieme item from the slice
		// Map all fields into the item
		for j := 0; j < item.NumField(); j++ {
			fieldTag := item.Type().Field(j).Tag.Get("csv") // Get the tag of the jeme field of the struct
			if fieldTag == "" {
				continue
			}
			rflFld := item.Field(j) // Get the reflected value of the field
			fieldPos, ok := header[fieldTag]
			if !ok {
				continue
			}
			valStr := record[fieldPos]        // Get the value from the record
			err := storeValue(valStr, rflFld) // Store the value in the reflected field
			if err != nil {
				return fmt.Errorf("(record %v to the slice %v):\n	==> %v", record, item, err)
			}
		}
	}
	// Set destRv to be sliceRv
	destRv.Set(sliceRv)
	return nil
}

// Set the value of the rflVal to valStr
// Make some parsing if needed
// @param valStr: the value, as a string, that we want to store
// @param rflVal: the reflected value where we want to store our value
// @return an error if one occure
func storeValue(valStr string, rflVal reflect.Value) error {
	switch rflVal.Kind() {
	case reflect.String:
		rflVal.SetString(valStr)
	case reflect.Int:
		// Parse the value to an int
		value, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Error while parsing int %v:\n	==> %v", valStr, err)

		}
		rflVal.SetInt(value)
	case reflect.Float64:
		// Parse the value to an float
		value, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return fmt.Errorf("Error while parsing float %v:\n	==> %v", valStr, err)
		}
		rflVal.SetFloat(value)
	}

	return nil
}
