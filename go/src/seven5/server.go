package seven5

import (
	"fmt"
	"log"
	"strings"
	"net/http"
	"strconv"
	"encoding/json"
)

//AddResource maps a resource to a part of the URL space.  
//name the part of the url space, which should be the name of the rousrce.  This should be
//plural for Indexers (cars, people, oxen) and singular for Finders (car, person, ox).  The name 
//will be converted to all lower case.  The name should not include any slashes or spaces as this
//will trigger armageddon and destroy all life on this planet.
func (self *SimpleHandler) AddResource(name string, r interface{}) {
	
	//sanity check
	_, index:=r.(Indexer)
	_, find:=r.(Finder)
	
	if !self.sanityCheckDisplayed && !index && !find {
		log.Printf("WARNING: You passed in a resource to AddResource which can never be accessed (%T)!", r);
		log.Printf("HINT: Maybe your methods require a pointer recevier and you passed just a receiver?");
		self.sanityCheckDisplayed=true
	}
	withSlashes := fmt.Sprintf("/%s/", strings.ToLower(name))
	
	self.resource[withSlashes] = r
	//log.Printf("added resource of type %T at %s", r, name)
	http.Handle(withSlashes, http.HandlerFunc(handlerWrapper))
}
//RemoveAllResources dumps all the resource mappings known to the Handler.  Useful in tests.
func (self *SimpleHandler) RemoveAllResources() {
	self.resource = make(map[string]interface{})
}

//Handle converts the parameters from the HTTP level request to our rest-level values
//and dumps the output on the response.  This is not used in tests, only when running
//against a real network.
func (self *SimpleHandler) Handle(c http.ResponseWriter, req *http.Request) {
	hdr := toSimpleMap(req.Header)
	qparams := toSimpleMap(map[string][]string(req.URL.Query()))
	json, err := self.Dispatch(req.Method, req.URL.Path, hdr, qparams)
	dumpOutput(c, json, err)
}

//SimpleHandler is the default implementation of the Handler interface that ignores multiple values for
//headers and query params because these are both rare and error-prone.  
type SimpleHandler struct {
	//resource maps names in URL space to objects that implement one or more of our rest interfaces
	resource map[string]interface{}
	//sanityCheckDisplayed is true if we have already displayed the "useless" warning
	sanityCheckDisplayed bool
}

//NewSimpleHandler creates a new SimpleHandler with an empty mapping.
func NewSimpleHandler() *SimpleHandler {
	return &SimpleHandler{resource: make(map[string]interface{})}
}

//Dispatch does the dirty work of finding a resource and calling it.
//It returns the value from the correct rest-level function or an error.
//It generates some errors itself if, for example a 404 or 501 is needed.
//I borrowed lots of ideas and inspiration from "github.com/Kissaki/rest2go"
func (self *SimpleHandler) Dispatch(method string, uriPath string, header map[string]string, 
	queryParams map[string]string) (string, *Error) {

	// try to get resource with full uri path
	someResource, ok := self.resource[uriPath]
	var id string
	var err error
	
	if !ok {
		// no resource found, thus check if the path is a resource + ID
		i := strings.LastIndex(uriPath, "/")
		if i == -1 {
			//log.Printf("No mapping found and no slash found in URIPath %s", uriPath)
			return NotFound()
		}
		// Move index to after slash as that’s where we want to split
		i++
		id = uriPath[i:]
		var uriPathParent string
		uriPathParent = uriPath[:i]
		someResource,ok = self.resource[uriPathParent]
		if !ok {
			//log.Printf("No mapping for URIPath-Parent %s", uriPathParent)
			return NotFound()
		}
	}
	switch method {
	case "GET":
		if len(id) == 0 {
			if resIndex, ok := someResource.(Indexer); ok {
				return resIndex.Index(header, queryParams)
			} else {
				//log.Printf("%T isn't an Indexer, returning NotImplemented", someResource)
				return NotImplemented()
			}
		} else {
			// Find by ID
			var num int
			if num, err = strconv.Atoi(id); err!=nil {
				return BadRequest("resource ids must be non-negative integers")
			}
			//resource id is a number, try to find it
			if resFind, ok := someResource.(Finder); ok {
				return resFind.Find(int32(num), header, queryParams)
			} else {
				return NotImplemented()
			}
		}
	}
	return "", &Error{http.StatusNotImplemented,"","Not implemented yet"}
}

//dumpOutput send the output to the calling client over the network.  Not used in tests,
//only when running against real network.
func dumpOutput(response http.ResponseWriter, json string, err *Error)  {
	if err!=nil && json!="" {
		log.Printf("ignoring json reponse (%d bytes) because also have error func %+v", len(json), err)
	}
	if err!=nil{
		if err.Location!="" {
			response.Header().Add("Location",err.Location)
		}
		http.Error(response, err.Message, err.StatusCode)
		return
	}
	if _,err:=response.Write([]byte(json)); err!=nil {
		log.Printf("error writing json response %s",err)
	}
	return
}

//BadRequest returns an error struct representing a 402, Bad Request HTTP response. This should be returned
//when the parameters passed the by client don't make sense.
func BadRequest(msg string) (string,*Error) {
	return "",&Error{http.StatusBadRequest, "", fmt.Sprintf("BadRequest - %s ", msg)}
}

//NoContent Returns an error struct representing a "succcess" in the sense of the protocol but 
//a semantic error of "empty".
func NoContent() (string,*Error) {
	return "",&Error{http.StatusNoContent, "", "No content"}
}

//NotFound Returns a 'these are not the droids you're looking for... move along.'
func NotFound() (string,*Error) {
	return "",&Error{http.StatusNotFound, "", "Not found"}
}

//NotImplemented returns an http 501.  This happens if we find a resource at the URL _but_ the 
//implementing struct doesn't have the correct type, for example /foobars/ is a known mapping
//but the struct does not implement Indexer.
func NotImplemented() (string,*Error) {
	return "",&Error{http.StatusNotImplemented, "", "Not implemented"}
}

//InternalErr is a convenience for returning a 501 when an error has been found at the go level.
func InternalErr(err error) (string,*Error) {
	return "",&Error{http.StatusInternalServerError, "", err.Error()}
}
//toSimpleMap converts an http level map with multiple strings as value to single string value.
func toSimpleMap(m map[string][]string) map[string]string {
	result :=make(map[string]string)
	for k,v := range m {
		result[k]=v[0]
	}
	return result
}

//JsonResult returns a json string from the supplied value or return an error (caused by the encoder)
//via the InternalErr function.  This is the normal path for functions that return Json values.
//pretty can be set to true for pretty-printed json.
func JsonResult(v interface{}, pretty bool) (string,*Error){
	var buff []byte
	var err error
	
	if pretty {
		buff, err = json.MarshalIndent(v, "", " ")
	} else {
		buff, err = json.Marshal(v)
	}
	if err!=nil {
		return InternalErr(err);
	}
	result := string(buff)
	return strings.Trim(result, " "),nil
	
}