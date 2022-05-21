package models

import (
	"context"
	"crypto/tls"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/astaxie/beego"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

type DbHandler struct {
	Distinct bool
	Limit    int
	Offset   int
	SortBy   string
	SortDir  string
	Search   string
	Filter   string
	BindVars map[string]interface{}
}

var (
	ZapLogger   *zap.Logger
	dbConnect   driver.Database
	DbQuery     string
	objectCache map[string]interface{}
)

func init() {
	dbHandler := DbHandler{}
	dbHandler.Limit = 10
	dbHandler.Offset = 0
	dbHandler.SortBy = "d._id"
	dbHandler.SortDir = "DESC"
	dbHandler.Search = ""

	dbConnect = connect()

}

func connect() driver.Database {
	// load database
	if dbConnect != nil {
		return dbConnect
	}
	connConfig := new(http.ConnectionConfig)
	dbURLs := strings.Split(beego.AppConfig.String("dburls"), ",")
	connConfig.Endpoints = dbURLs
	enabledHTTPS, _ := beego.AppConfig.Bool("EnableHTTPS")
	if enabledHTTPS {
		connConfig.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, err := http.NewConnection(*connConfig)

	if err != nil {
		panic("Database connection error")
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(beego.AppConfig.String("dbuser"), beego.AppConfig.String("dbpass")),
	})
	//	"github.com/labstack/gommon/log"
	// log.Info(beego.AppConfig.String("dbuser"), beego.AppConfig.String("dbpass"))
	// fmt.Println(beego.AppConfig.String("dbuser"), beego.AppConfig.String("dbpass"))

	if err != nil {
		panic("Error when creating client")
	}
	ctx := context.Background()
	db, err := c.Database(ctx, beego.AppConfig.String("dbdb"))
	if err != nil {
		panic("Error when getting database")
	}

	return db
}
func (db *DbHandler) GetConnection() driver.Database {
	return connect()
}

func (db *DbHandler) GetView(viewName string, docKey ...string) ([]interface{}, int64, error) {
	/**
	FOR doc IN selectAnyRegions
	SEARCH
	    STARTS_WITH(doc.name, @search) OR
	    PHRASE(doc.name, @search, 'text_en') OR
	    ANALYZER(doc.name IN TOKENS(@search, 'text_en'), 'text_en') OR
	    ANALYZER(STARTS_WITH(doc.name, @search), 'text_en')
	SORT doc.name ASC
	LIMIT 10
	RETURN doc
			**/
	ctx := driver.WithQueryFullCount(context.Background())
	bindVars := make(map[string]interface{})
	query := "FOR doc IN " + viewName

	searchString := db.Search
	var docKeyVal string
	if db.Search != "" {
		if len(docKey) == 0 {
			docKeyVal = "_id"
		} else {
			docKeyVal = docKey[0]
		}

		searchString = strings.TrimSpace(searchString)
		searchString = strings.ToUpper(searchString)
		// @link https://github.com/arangodb/arangodb/issues/7825#issuecomment-449356718
		// query += " SEARCH STARTS_WITH(doc." + docKeyVal + ", @search)"
		// query += " SEARCH ANALYZER(STARTS_WITH(doc." + docKeyVal + ", @search), 'text_en')"
		// query += " SEARCH ANALYZER(doc.name IN TOKENS(@search, 'text_en'), 'text_en')"
		query += ` SEARCH`
		query += ` STARTS_WITH(doc.` + docKeyVal + `, @search) OR`
		query += ` PHRASE(doc.` + docKeyVal + `, @search, 'text_en') OR`
		query += ` ANALYZER(doc.` + docKeyVal + ` IN TOKENS(@search, 'text_en'), 'text_en') OR`
		query += ` ANALYZER(STARTS_WITH(doc.` + docKeyVal + `, @search), 'text_en')`
		bindVars["search"] = `%` + searchString + `%`
	}
	if db.SortBy == "" {
		query += " SORT TFIDF(doc) DESC"
	} else {
		query += " SORT doc." + db.SortBy + " " + db.SortDir
	}
	if db.Limit > 0 {
		query += " LIMIT " + strconv.Itoa(db.Limit)
	}
	query += " RETURN doc"

	cursor, err := dbConnect.Query(ctx, query, bindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, 0, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}
	stat := cursor.Statistics()

	return collection, stat.FullCount(), nil
}

// GetObject https://www.arangodb.com/docs/stable/drivers/go-example-requests.html#reading-a-document-from-a-collection
func (db *DbHandler) GetObject(collectionName string, docKey string /*, docType ...interface{}*/) (interface{}, error) {
	if docKey == "" {
		return nil, errors.New("Missing docKey")
	}
	convert, err := strconv.Atoi(docKey)
	if err != nil {
		return nil, err
	}
	if convert == 0 {
		return nil, errors.New("Missing docKey")
	}
	revert := strconv.Itoa(convert)
	if revert == "" {
		return nil, errors.New("Missing docKey")
	}

	// get object cache
	// cache := db.getObjectCache(collectionName, revert)
	// if cache != nil {
	// 	return cache, nil
	// }

	ctx := context.Background()
	// col, err := dbConnect.Collection(ctx, collectionName)
	// if err != nil {
	// 	return nil, err
	// }
	// ERROR: https://github.com/arangodb/go-driver/issues/174
	// var doc interface{}
	// meta, err := col.ReadDocument(nil, revert, &doc)
	// if err != nil {
	// 	return meta, nil, err
	// }

	query := "FOR doc IN " + collectionName
	if docKey != "" {
		query += " FILTER doc._key=='" + revert + "'"
	} else if db.Filter != "" {
		query += " " + db.Filter
	}
	query += " LIMIT 1"
	query += " RETURN doc"

	cur, err := dbConnect.Query(ctx, query, nil)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}
	defer cur.Close()

	var doc interface{}
	// if docType != nil && len(docType) > 0 {
	// 	doc = docType[0]
	// }
	_, err = cur.ReadDocument(ctx, &doc)
	if driver.IsNoMoreDocuments(err) {
		return nil, err
	} else if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}

	// set object cache
	// db.setObjectCache(collectionName, revert, doc)

	return doc, nil
}

func (db *DbHandler) GetObjectByQuery(query string /*, docType ...interface{}*/) (interface{}, error) {
	ctx := context.Background()
	cur, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}
	defer cur.Close()

	var doc interface{}
	// if docType != nil && len(docType) > 0 {
	// 	doc = docType[0]
	// }
	_, err = cur.ReadDocument(ctx, &doc)
	if driver.IsNoMoreDocuments(err) {
		return nil, err
	} else if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}

	return doc, nil
}

func (db *DbHandler) GetCollectionByQuery(query string /*, docType ...interface{}*/) ([]interface{}, error) {
	ctx := context.Background()
	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		// if docType != nil && len(docType) > 0 {
		// 	doc = docType[0]
		// }
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}

	return collection, nil
}

func (db *DbHandler) GetCollection(collectionName string) ([]interface{}, error) {
	ctx := context.Background()
	var query string
	if DbQuery != "" {
		query = DbQuery
	} else {
		query = "FOR d IN " + collectionName
		if db.Filter != "" {
			query += " " + db.Filter
		}
		if db.SortBy != "" {
			query += " SORT d." + db.SortBy + " " + db.SortDir
		}
		if db.Limit > 0 && db.Offset == 0 {
			query += " LIMIT " + strconv.Itoa(db.Limit)
		}
		if db.Limit > 0 && db.Offset > 0 {
			query += " LIMIT " + strconv.Itoa(db.Offset) + "," + strconv.Itoa(db.Limit)
		}
		query += " RETURN d"
	}

	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}

	return collection, nil
}

func (db *DbHandler) GetInfiniteCollection(collectionName string) ([]interface{}, bool, error) {
	ctx := context.Background()
	var query string
	if DbQuery != "" {
		query = DbQuery
	} else {
		query = "FOR d IN " + collectionName
		if db.Filter != "" {
			query += " " + db.Filter
		}
		if db.SortBy != "" {
			query += " SORT d." + db.SortBy + " " + db.SortDir
		}
		if db.Limit > 0 && db.Offset == 0 {
			query += " LIMIT " + strconv.Itoa(db.Limit)
		}
		if db.Limit > 0 && db.Offset > 0 {
			query += " LIMIT " + strconv.Itoa(db.Offset) + "," + strconv.Itoa(db.Limit)
		}
		query += " RETURN d"
	}

	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, false, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}

	// for select2 pagination
	var more bool
	if db.Limit > 0 && len(collection) == db.Limit {
		checkMore := DbHandler{
			Limit:    db.Limit,
			Offset:   db.Offset + db.Limit,
			SortBy:   db.SortBy,
			SortDir:  db.SortDir,
			Search:   db.Search,
			Filter:   db.Filter,
			BindVars: db.BindVars,
		}
		col, err := checkMore.GetCollection(collectionName)
		if err == nil && len(col) > 0 {
			more = true
		}
	}

	return collection, more, nil
}

func (db *DbHandler) GetCollectionWithCount(collectionName string) ([]interface{}, int64, error) {
	ctx := driver.WithQueryFullCount(context.Background())
	query := "FOR d IN " + collectionName
	if db.Filter != "" {
		query += " " + db.Filter
	}
	if db.SortBy == "" {
		db.SortBy = "_id"
	}
	if db.SortDir == "" {
		db.SortDir = "desc"
	}
	query += " SORT d." + db.SortBy + " " + db.SortDir
	if db.Limit > 0 && db.Offset == 0 {
		query += " LIMIT " + strconv.Itoa(db.Limit)
	}
	if db.Limit > 0 && db.Offset > 0 {
		query += " LIMIT " + strconv.Itoa(db.Offset) + "," + strconv.Itoa(db.Limit)
	}
	query += " RETURN d"

	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, 0, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}

	stat := cursor.Statistics()

	return collection, stat.FullCount(), nil
}

func (db *DbHandler) GetCollectionGraph(withCollections []string, edgeName string, startNodeId string, direction ...string) ([]interface{}, error) {
	ctx := context.Background()
	var directionVar string
	if len(direction) == 0 {
		directionVar = "OUTBOUND"
	} else {
		directionVar = direction[0]
	}
	if db.SortBy == "" {
		db.SortBy = "_id"
	}
	if db.SortDir == "" {
		db.SortDir = "desc"
	}

	query := `WITH ` + strings.Join(withCollections, ",")
	query += " FOR d,e IN " + directionVar + " '" + startNodeId + "' " + edgeName
	if db.Filter != "" {
		query += " " + db.Filter
	} else {
		query += " OPTIONS {bfs: true, uniqueVertices: 'global'}"
	}
	query += " SORT d." + db.SortBy + " " + db.SortDir
	if db.Limit > 0 && db.Offset == 0 {
		query += " LIMIT " + strconv.Itoa(db.Limit)
	}
	if db.Limit > 0 && db.Offset > 0 {
		query += " LIMIT " + strconv.Itoa(db.Offset) + "," + strconv.Itoa(db.Limit)
	}
	if db.Distinct == true {
		query += " RETURN DISTINCT d"
	} else {
		query += " RETURN d"
	}

	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}

	return collection, nil
}

func (db *DbHandler) GetCollectionGraphWithCount(withCollections []string, edgeName string, startNodeId string, direction ...string) ([]interface{}, int64, error) {
	ctx := driver.WithQueryFullCount(context.Background())
	var directionVar string
	if len(direction) == 0 {
		directionVar = "OUTBOUND"
	} else {
		directionVar = direction[0]
	}

	query := `WITH ` + strings.Join(withCollections, ",")
	query += " FOR d IN " + directionVar + " '" + startNodeId + "' " + edgeName
	if db.Filter != "" {
		query += db.Filter
	} else {
		query += " OPTIONS {bfs: true, uniqueVertices: 'global'}"
	}
	if db.SortBy == "" {
		db.SortBy = "d._id"
	}
	if db.SortDir == "" {
		db.SortDir = "desc"
	}
	query += " SORT " + db.SortBy + " " + db.SortDir
	if db.Limit > 0 && db.Offset == 0 {
		query += " LIMIT " + strconv.Itoa(db.Limit)
	}
	if db.Limit > 0 && db.Offset > 0 {
		query += " LIMIT " + strconv.Itoa(db.Offset) + "," + strconv.Itoa(db.Limit)
	}
	query += " RETURN d"

	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, 0, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}
	stat := cursor.Statistics()

	return collection, stat.FullCount(), nil
}

func (db *DbHandler) GetCollectionByQueryWithCount(query string) ([]interface{}, int64, error) {
	ctx := driver.WithQueryFullCount(context.Background())
	cursor, err := dbConnect.Query(ctx, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Error(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Error(`db.BindVars: ` + strDebug)
		return nil, 0, err
	}
	defer cursor.Close()

	var collection []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Error(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Error(`db.BindVars: ` + strDebug)
			continue
		}
		if doc != nil {
			collection = append(collection, doc)
		}
	}
	stat := cursor.Statistics()

	return collection, stat.FullCount(), nil
}

func (db *DbHandler) SetLimit(limit int) {
	db.Limit = limit
}

func (db *DbHandler) SetOffset(offset int) {
	db.Offset = offset
}

func (db *DbHandler) SetSortBy(sortby string) {
	db.SortBy = sortby
}

func (db *DbHandler) SetSortDir(sortdir string) {
	db.SortDir = sortdir
}

func (db *DbHandler) SetSearch(search string) {
	db.Search = search
}

func (db *DbHandler) SetQuery(query string) {
	DbQuery = query
}

func (db *DbHandler) SaveDocument(collection string, data interface{}) (interface{}, error) {
	ctx := context.Background()
	col, err := db.GetConnection().Collection(ctx, collection)
	if err != nil {
		ZapLogger.Error(err.Error())
		ZapLogger.Error(`collection: ` + collection)
		var strDebug string
		strDebug = spew.Sdump(data)
		ZapLogger.Error(`data: ` + strDebug)
		return nil, err
	}

	meta, err := col.CreateDocument(ctx, data)
	if err != nil {
		ZapLogger.Error(err.Error())
		return nil, err
	}
	// Document must exists now
	if found, err := col.DocumentExists(ctx, meta.Key); err != nil {
		ZapLogger.Error(err.Error())
		return nil, err
	} else if !found {
		ZapLogger.Error("DocumentExists returned false for '" + meta.Key + "', expected true")
		return nil, errors.New("DocumentExists returned false for " + meta.Key)
	}

	// Read and return document
	var readDoc map[string]interface{}
	// ERROR: bug in arangodb's driver
	// @link https://github.com/arangodb/go-driver/issues/174
	if _, err := col.ReadDocument(ctx, meta.Key, &readDoc); err != nil {
		ZapLogger.Error("Failed to read document '" + meta.Key + "': " + err.Error())
		return nil, err
	}

	// db.setObjectCache(collection, meta.Key, readDoc)

	return readDoc, nil
}

// SaveEdge presents abstraction to save edge
// @link https://godoc.org/github.com/arangodb/go-driver#ex-package--CreateGraph
func (db *DbHandler) SaveEdge(collection string, graphName string, fromCollection string, toCollection string, document interface{}) (interface{}, error) {
	ctx := context.Background()
	// define the edgeCollection to store the edges
	var edgeDefinition driver.EdgeDefinition
	edgeDefinition.Collection = collection

	// define a set of collections where an edge is going out...
	edgeDefinition.From = []string{fromCollection}

	// repeat this for the collections where an edge is going into
	edgeDefinition.To = []string{toCollection}

	var graph driver.Graph
	var err error
	doesGraphExist, _ := dbConnect.GraphExists(ctx, graphName)
	if doesGraphExist {
		// get a graph
		graph, err = dbConnect.Graph(ctx, graphName)
		if err != nil {
			ZapLogger.Error(err.Error())
			return nil, err
		}
	} else {
		// now it's possible to create a graph
		var options driver.CreateGraphOptions
		options.EdgeDefinitions = []driver.EdgeDefinition{edgeDefinition}

		graph, err = dbConnect.CreateGraph(ctx, graphName, &options)
		if err != nil {
			ZapLogger.Error(err.Error())
			return nil, err
		}
	}

	/* Check */
	val := reflect.ValueOf(document)
	docMap := make(map[string]interface{})
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		docMap[typeField.Name] = valueField.Interface()

	}
	query := "FOR d IN " + collection
	query += " FILTER d._from==@from"
	query += " AND d._to==@to"
	query += " RETURN d"
	bindVars := make(map[string]interface{})
	bindVars["from"] = docMap["From"].(string)
	bindVars["to"] = docMap["To"].(string)
	db.BindVars = bindVars

	existing, errExisting := db.GetCollectionByQuery(query)
	if driver.IsNoMoreDocuments(errExisting) {
		// go below
	} else if existing != nil {
		existingDoc := make(map[string]interface{})
		for _, item := range existing {
			existingDoc = item.(map[string]interface{})
			break
		}
		updated, err := db.UpdateObject(collection, existingDoc["_key"].(string), document)
		if err != nil {
			ZapLogger.Error(err.Error())
			return nil, err
		}

		return updated, nil
	} else if errExisting != nil {
		return nil, errExisting
	}

	// add edge
	ec, _, err := graph.EdgeCollection(ctx, collection)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(collection)
		ZapLogger.Error(`collection: ` + strDebug)
		return nil, err
	}

	meta, err := ec.CreateDocument(ctx, document)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(document)
		ZapLogger.Error(`document: ` + strDebug)
		return nil, err
	}

	// Document must exists now
	if found, err := ec.DocumentExists(nil, meta.Key); err != nil {
		ZapLogger.Error("DocumentExists failed for '" + meta.Key + "': " + err.Error())
		return nil, err
	} else if !found {
		ZapLogger.Error("DocumentExists returned false for '" + meta.Key + "', expected true")
		return nil, errors.New("DocumentExists returned false for '" + meta.Key + "', expected true")
	}
	// Read edge
	// dataType := reflect.TypeOf(document)
	// readDoc := reflect.Zero(dataType)
	var docMap2 map[string]interface{}
	_ /*getDoc*/, err2 := ec.ReadDocument(ctx, meta.Key, &docMap2)
	if err2 != nil {
		ZapLogger.Error("Failed to read edge '" + meta.Key + "': " + err2.Error())
		return nil, err2
	}

	return docMap2, nil
}

func (db *DbHandler) UpdateObject(collection string, docKey string, update interface{}) (interface{}, error) {
	ctx := context.Background()
	col, err := db.GetConnection().Collection(ctx, collection)
	if err != nil {
		ZapLogger.Error(err.Error())
		ZapLogger.Error("collection " + collection)
		ZapLogger.Error("docKey " + docKey)
		var strDebug string
		strDebug = spew.Sdump(update)
		ZapLogger.Error(`update: ` + strDebug)
		return nil, err
	}
	meta, err2 := col.UpdateDocument(ctx, docKey, update)
	if err2 != nil {
		ZapLogger.Error(err2.Error())
		ZapLogger.Error("collection: " + collection)
		ZapLogger.Error("docKey: " + docKey)
		var strDebug string
		strDebug = spew.Sdump(update)
		ZapLogger.Error(`update: ` + strDebug)
		return nil, err
	}
	getObject, err3 := db.GetObject(collection, meta.Key)
	if err3 != nil {
		ZapLogger.Error(err3.Error())
		return nil, err3
	}

	return getObject, nil
}

func (db *DbHandler) RemoveObject(collection string, docKey string) (interface{}, error) {
	if docKey == "" {
		return nil, errors.New("Missing docKey")
	}
	convert, err := strconv.Atoi(docKey)
	if err != nil {
		return nil, err
	}
	if convert == 0 {
		return nil, errors.New("Missing docKey")
	}
	revert := strconv.Itoa(convert)
	if revert == "" {
		return nil, errors.New("Missing docKey")
	}
	ctx := context.Background()
	col, err := dbConnect.Collection(ctx, collection)
	if err != nil {
		ZapLogger.Error(err.Error())
		return nil, err
	}
	meta, err := col.RemoveDocument(ctx, revert)
	if err != nil {
		ZapLogger.Error(err.Error())
		ZapLogger.Error(`collection: ` + collection)
		ZapLogger.Error(`docKey: ` + docKey)
		return nil, err
	}

	return meta, nil
}

func (db *DbHandler) GetObjectInboundGraph(withCollections []string, edgeName string, startNodeId string /*, docType ...interface{}*/) (interface{}, error) {
	ctx := context.Background()

	query := `WITH ` + strings.Join(withCollections, ",")
	query += " FOR v IN 1..1 INBOUND @startNodeId @@edgeName"
	if db.Filter != "" {
		query += " " + db.Filter
	} else {
		query += " OPTIONS {bfs: true, uniqueVertices: 'global'}"
	}
	query += " RETURN v"

	bindVars := make(map[string]interface{})
	bindVars["startNodeId"] = startNodeId
	bindVars["@edgeName"] = edgeName
	if db.BindVars != nil && len(db.BindVars) > 0 {
		for key, val := range db.BindVars {
			bindVars[key] = val
		}
	}
	cursor, err := dbConnect.Query(ctx, query, bindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Info(`query: ` + strDebug)
		strDebug = spew.Sdump(bindVars)
		ZapLogger.Info(`bindVars: ` + strDebug)
		return nil, err
	}
	defer cursor.Close()

	var doc interface{}
	// if docType != nil && len(docType) > 0 {
	// 	doc = docType[0]
	// }
	for {
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Info(`query: ` + strDebug)
			strDebug = spew.Sdump(bindVars)
			ZapLogger.Info(`bindVars: ` + strDebug)
			continue
		}
	}

	return doc, nil
}

func (db *DbHandler) GetObjectOutboundGraph(withCollections []string, edgeName string, startNodeId string /*, docType ...interface{}*/) (interface{}, error) {
	ctx := context.Background()

	query := `WITH ` + strings.Join(withCollections, ",")
	query += " FOR v IN 1..1 OUTBOUND @startNodeId @@edgeName"
	if db.Filter != "" {
		query += " " + db.Filter
	}
	query += " RETURN v"

	bindVars := make(map[string]interface{})
	bindVars["startNodeId"] = startNodeId
	bindVars["@edgeName"] = edgeName
	if db.BindVars != nil && len(db.BindVars) > 0 {
		for key, val := range db.BindVars {
			bindVars[key] = val
		}
	}
	cursor, err := dbConnect.Query(ctx, query, bindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Info(`query: ` + strDebug)
		strDebug = spew.Sdump(bindVars)
		ZapLogger.Info(`bindVars: ` + strDebug)
		return nil, err
	}
	defer cursor.Close()

	var doc interface{}
	// if docType != nil && len(docType) > 0 {
	// 	doc = docType[0]
	// }
	for {
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Info(`query: ` + strDebug)
			strDebug = spew.Sdump(bindVars)
			ZapLogger.Info(`bindVars: ` + strDebug)
			continue
		}
	}

	return doc, nil
}

func (db *DbHandler) RemoveObjectGraph(withCollections []string, graphName string, edgeName string, vertexName string, startNodeId string, endNodeKey string, direction ...string) error {
	ctx := context.Background()
	// remove edge and graph
	// https://stackoverflow.com/a/56818884/1246646
	dir := "OUTBOUND"
	if direction != nil && len(direction) > 0 {
		dir = direction[0]
	}

	query := `WITH ` + strings.Join(withCollections, ",")
	query += " LET keys = (FOR v, e IN 1..1 " + dir + " @startNodeId GRAPH @graphName REMOVE e._key IN @@edgeName)"
	query += " REMOVE @endNodeKey IN @@vertexName"

	bindVars := make(map[string]interface{})
	bindVars["graphName"] = graphName
	bindVars["@edgeName"] = edgeName
	bindVars["@vertexName"] = vertexName
	bindVars["startNodeId"] = startNodeId
	bindVars["endNodeKey"] = endNodeKey

	cursor, err := db.GetConnection().Query(ctx, query, bindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Info(`query: ` + strDebug)
		strDebug = spew.Sdump(bindVars)
		ZapLogger.Info(`bindVars: ` + strDebug)
		return err
	}
	defer cursor.Close()

	return nil
}

func (db *DbHandler) RemoveEdges(withCollections []string, graphName, edgeName, startNodeId string, direction ...string) error {
	if graphName == "" || edgeName == "" || startNodeId == "" {
		return errors.New("Missing parameter")
	}
	ctx := context.Background()
	// remove edge and graph
	// https://stackoverflow.com/a/56818884/1246646
	dir := "OUTBOUND"
	if direction != nil && len(direction) > 0 {
		dir = direction[0]
	}
	query := `WITH ` + strings.Join(withCollections, ",")
	query += " FOR v, e IN 1..1 " + dir + " @startNodeId GRAPH @graphName REMOVE e._key IN @@edgeName OPTIONS { ignoreErrors: true }"

	bindVars := make(map[string]interface{})
	bindVars["startNodeId"] = startNodeId
	bindVars["graphName"] = graphName
	bindVars["@edgeName"] = edgeName

	cursor, err := db.GetConnection().Query(ctx, query, bindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Info(`query: ` + strDebug)
		strDebug = spew.Sdump(bindVars)
		ZapLogger.Info(`bindVars: ` + strDebug)
		return err
	}
	defer cursor.Close()

	return nil
}

func (db *DbHandler) GetCountByQuery(query string) (float64, error) {
	if query == "" {
		ZapLogger.Error("Missing query")
		return 0, errors.New("Missing query")
	}

	cursor, err := db.GetConnection().Query(nil, query, db.BindVars)
	if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Info(`query: ` + strDebug)
		strDebug = spew.Sdump(db.BindVars)
		ZapLogger.Info(`db.BindVars: ` + strDebug)
		return 0, err
	}
	defer cursor.Close()

	var doc float64
	for {
		_, err := cursor.ReadDocument(nil, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			ZapLogger.Error(err.Error())
			var strDebug string
			strDebug = spew.Sdump(query)
			ZapLogger.Info(`query: ` + strDebug)
			strDebug = spew.Sdump(db.BindVars)
			ZapLogger.Info(`db.BindVars: ` + strDebug)
			continue
		}
	}

	return doc, nil
}

func (db *DbHandler) getObjectCache(collectionName, docKey string) interface{} {
	if objectCache == nil {
		return nil
	}
	if objectCache[collectionName] == nil {
		return nil
	}
	objectCacheMap := objectCache[collectionName].(map[string]interface{})
	if objectCacheMap[docKey] != nil && len(objectCacheMap[docKey].(map[string]interface{})) > 0 {
		return objectCacheMap[docKey]
	}

	return nil
}

func (db *DbHandler) setObjectCache(collectionName, docKey string, value interface{}) {
	if objectCache == nil {
		objectCache = make(map[string]interface{})
	}
	if objectCache[collectionName] == nil {
		objectCache[collectionName] = make(map[string]interface{})
	}
	objectCacheMap := make(map[string]interface{})
	objectCacheMap[docKey] = value
	objectCache[collectionName] = objectCacheMap
}

func (db *DbHandler) GetObjectByField(collectionName, fieldName, value string) (interface{}, error) {
	if collectionName == "" {
		return nil, errors.New("Missing collectionName")
	}
	if fieldName == "" {
		return nil, errors.New("Missing fieldName")
	}
	if value == "" {
		return nil, errors.New("Missing fieldValue")
	}

	bindVars := make(map[string]interface{})
	query := "FOR doc IN @@collectionName"
	bindVars["@collectionName"] = collectionName

	query += " FILTER doc.@fieldName == @value"
	bindVars["fieldName"] = fieldName
	bindVars["value"] = value
	query += " LIMIT 1"
	query += " RETURN doc"

	db.BindVars = bindVars

	return db.GetObjectByQuery(query)
}

// MigrationTimestamp is used to flag delta difference between ArangoDB and Postgre
func MigrationTimestamp() string {
	current_time := time.Now().UTC()

	return current_time.Format(time.RFC3339Nano)
}
