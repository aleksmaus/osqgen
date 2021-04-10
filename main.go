package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
)

type Column struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type Table struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Platforms   []string `json:"platforms"`
	Columns     []Column `json:"columns"`
}

type ColumnInfo struct {
	TableName string
	Column    Column
}

type ColumnDuplicate struct {
	Original  ColumnInfo
	Duplicate ColumnInfo
}

const (
	cmdFields = "fields"
)

var (
	errUnsupportedCommand = errors.New("unsupported command")
)

var schemaFileName string

func exitOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

func main() {
	var err error

	flag.Usage = func() {
		fmt.Println("Usage: osqgen [options] command")
		flag.PrintDefaults()
	}

	flag.StringVar(&schemaFileName, "schema", "", "Schema file name")

	flag.Parse()

	if schemaFileName == "" || len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	command := flag.Args()[0]

	switch command {
	case cmdFields:
		err = execFieldsCommand()
	default:
		err = errUnsupportedCommand
	}

	exitOnError(err)
}

func loadSchema(schemaFileName string) (tables []Table, err error) {
	f, err := os.Open(schemaFileName)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	d := json.NewDecoder(f)
	err = d.Decode(&tables)
	return
}

func convergeToESType(t string) string {
	switch t {
	case "integer", "unsigned_bigint", "bigint":
		return "long"
	}
	return t
}

func printDuplicateColumns(dupColumnsMap map[string][]ColumnDuplicate) {

	keys := make([]string, 0, len(dupColumnsMap))
	for k := range dupColumnsMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, colName := range keys {
		dupColumns := dupColumnsMap[colName]
		c := len(dupColumns)
		for _, dup := range dupColumns {
			fmt.Printf("%d: %s.%s[%s] %s.%s[%s]\n",
				c,
				dup.Original.TableName, dup.Original.Column.Name, dup.Original.Column.Type,
				dup.Duplicate.TableName, dup.Duplicate.Column.Name, dup.Duplicate.Column.Type,
			)
		}
	}
}

func execFieldsCommand() error {
	tables, err := loadSchema(schemaFileName)
	if err != nil {
		return err
	}

	columns := make(map[string]ColumnInfo)

	dupColumnsMap := make(map[string][]ColumnDuplicate)

	total := 0
	for _, table := range tables {
		for _, column := range table.Columns {
			total += 1
			column.Type = convergeToESType(column.Type)
			colInfo, ok := columns[column.Name]
			if ok {
				if colInfo.Column.Type != column.Type {
					dupColumns, found := dupColumnsMap[column.Name]
					if !found {
						dupColumns = make([]ColumnDuplicate, 0)
					}
					dupColumns = append(dupColumns, ColumnDuplicate{
						Original:  colInfo,
						Duplicate: ColumnInfo{table.Name, column},
					})
					dupColumnsMap[column.Name] = dupColumns
				}
			} else {
				columns[column.Name] = ColumnInfo{table.Name, column}
			}
		}
	}

	// printDuplicateColumns(dupColumnsMap)

	// fmt.Printf("Total: %d\n", total)

	return generateFields(os.Stdout, columns, dupColumnsMap)
}
