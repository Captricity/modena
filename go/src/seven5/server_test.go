package seven5

import (
	"testing"
	"fmt"
)


/*-------------------------------------------------------------------------*/
type ExampleIndexer_wrongNoDoc struct {
}
func (STATELESS *ExampleIndexer_wrongNoDoc) Index(headers map[string]string,
		queryParams map[string]string) (string,*Error)  {
		return "{}",nil
}
/*-------------------------------------------------------------------------*/
type ExampleIndexer_correct struct {
}

func (STATELESS *ExampleIndexer_correct) Index(headers map[string]string,queryParams map[string]string) (string,*Error)  {
		return "{}",nil
}
func (STATELESS *ExampleIndexer_correct) IndexDoc() (string,string,string) {
	return "","",""
}

/*-------------------------------------------------------------------------*/
type Ox struct {
	Id int32
	IsLarge bool
}

type ExampleFinder_correct struct {
}

func (STATELESS *ExampleFinder_correct)	Find(id int32, headers map[string]string, 
	queryParams map[string]string) (string,*Error) {
		switch (id) {
		case 0,1:
			return JsonResult(&Ox{id,true},false);
		}	
		return NotFound();
}
func (STATELESS *ExampleFinder_correct)	FindDoc() string {
	return "How can you lose an ox?"
}
func (STATELESS *ExampleFinder_correct) FindFields() map[string]*FieldDoc {
	return map[string]*FieldDoc{
		"IsLarge": &FieldDoc{false, "Set to `true` if this a _really_ big ox!"},
	};
}
/*-------------------------------------------------------------------------*/
func verifyErrorCode(T *testing.T, err *Error, expected int, msg string) {
	if err==nil {
		T.Errorf("expected error with code %d but got no error at all (%s)", expected, msg)
		return
	}
	if err.StatusCode != expected {
		T.Errorf("expected error code %d but got %d (%s)", expected, err.StatusCode, msg)
	}
}

func verifyDispatchError(T *testing.T, errorMap map[string]int) {
	for k,v := range errorMap {
		json, err := CurrentHandler.Dispatch("GET",k, emptyMap, emptyMap)
		if json!="" {
			T.Errorf("expected no json result on an error : GET %s",k)
			continue
		}
		verifyErrorCode(T,err,v, fmt.Sprintf("GET %s",k))
	}
}

func verifyNoError(T *testing.T, json string, err *Error, msg string){
	if err!=nil || json=="" {
		T.Fatalf("didn't expect error on %s but got %+v",msg, err);
	}
}
func verifyJsonContent(T *testing.T, json string, expected string, msg string){
	if json!=expected {
		T.Fatalf("expected json of '%s' but got '%s' from %s",expected, json,msg);
	}
}

/*-------------------------------------------------------------------------*/
var emptyMap = make(map[string]string)


func TestResourceMappingForIndex(T *testing.T) {
	CurrentHandler.RemoveAllResources()
		
	CurrentHandler.AddResource("oxen",&ExampleIndexer_wrongNoDoc{})
	CurrentHandler.AddResource("people",&ExampleIndexer_correct{})
	
	errorMap :=map[string]int{
		"oxen": 404,
		"/oxen": 404,
		"/oxen/": 501,
		"/people": 404,
		"people": 404,
	}
	
	verifyDispatchError(T, errorMap)
	
	json, err := CurrentHandler.Dispatch("GET","/people/", emptyMap, emptyMap)
	verifyNoError(T,json,err,"GET /people/")
	verifyJsonContent(T,json,"{}", "GET /people/")	
}

func TestResourceMappingForFinder(T *testing.T) {
	CurrentHandler.RemoveAllResources()

	CurrentHandler.AddResource("ox",&ExampleFinder_correct{})
	CurrentHandler.AddResource("person",&ExampleIndexer_wrongNoDoc{})


	errorMap :=map[string]int{
		"/oxen/": 404,
		"/oxen/123": 404,
		"/people/": 404,
		"/people/123": 404, 
		"/person/123": 501, //not implemented properly
		"/ox/123": 404, //too large an id
	}

	verifyDispatchError(T, errorMap)

	json, err := CurrentHandler.Dispatch("GET","/ox/0", emptyMap, emptyMap)
	verifyNoError(T,json,err, "GET /ox/0")
	verifyJsonContent(T,json,"{\"Id\":0,\"IsLarge\":true}", "GET /ox/0")	
}
