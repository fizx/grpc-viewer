package grpcviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	_ "embed"

	"github.com/hoisie/mustache"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MethodHandler = func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error)

type wrapper struct {
	inner    http.Handler
	server   *grpc.Server
	metadata map[string]interface{}
	impls    map[string]interface{}
	handlers map[string]MethodHandler
	inTypes  map[string]reflect.Type
}

type Both interface {
	http.Handler
	grpc.ServiceRegistrar
}

//go:embed template.html
var indexTemplate string

func NewServer() Both {
	server := grpc.NewServer()
	inner := server
	metadata := make(map[string]interface{})
	impls := make(map[string]interface{})
	handlers := make(map[string]MethodHandler)
	inTypes := make(map[string]reflect.Type)
	return &wrapper{inner, server, metadata, impls, handlers, inTypes}
}

func (wrapper *wrapper) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	wrapper.impls[desc.ServiceName] = impl
	formatter := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: true}
	info := make(map[string]interface{})
	ht := reflect.TypeOf(desc.HandlerType).Elem()
	for i := 0; i < ht.NumMethod(); i++ {
		m := ht.Method(i)
		minfo := make(map[string]interface{})
		if !strings.HasPrefix(m.Name, "mustEmbed") {
			// info["impl"] = impl
			for _, dm := range desc.Methods {
				if dm.MethodName == m.Name {
					wrapper.handlers[desc.ServiceName+"/"+m.Name] = dm.Handler
				}
			}
			info[m.Name] = minfo
			minfo["Name"] = m.Name
			typeIn := m.Type.In(1).Elem()
			wrapper.inTypes[desc.ServiceName+"/"+m.Name] = typeIn
			minfo["TypeIn"] = fmt.Sprintf("%v", typeIn)
			minfo["ExampleIn"] = formatter.Format(reflect.New(typeIn).Interface().(protoreflect.ProtoMessage))
		}
	}
	wrapper.metadata[desc.ServiceName] = info
	wrapper.server.RegisterService(desc, impl)
}

type readerCloser struct {
	reader io.Reader
	closer io.Closer
}

func (r *readerCloser) Read(dest []byte) (int, error) {
	return r.reader.Read(dest)
}
func (r *readerCloser) Close() error {
	return r.closer.Close()
}

func (wrapper *wrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := os.ReadFile("template.html")
		if err == nil {
			indexTemplate = string(tmpl)
		}
		vars := make(map[string]interface{})
		bytes, _ := json.Marshal(wrapper.metadata)
		vars["json"] = string(bytes)
		w.Write([]byte(mustache.Render(indexTemplate, vars)))
	} else if r.Method == "POST" && r.ProtoMajor == 1 {
		paths := strings.Split(r.URL.Path, "/")
		if len(paths) < 3 {
			w.WriteHeader(400)
			w.Write([]byte("No path segments\n"))
			return
		}
		s := paths[1]
		m := paths[2]
		impl, ok := wrapper.impls[s]
		if !ok {
			w.WriteHeader(400)
			w.Write([]byte("No service for " + s))
			return
		}
		println(s + "/" + m)
		handler, ok := wrapper.handlers[s+"/"+m]
		if !ok {
			w.WriteHeader(400)
			w.Write([]byte("No method for " + m))
			return
		}
		typ := wrapper.inTypes[s+"/"+m]
		msg := reflect.New(typ).Interface().(proto.Message)
		input, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		_ = protojson.Unmarshal(input, msg)
		dec := func(i interface{}) error {
			proto.Merge(i.(proto.Message), msg)
			return nil
		}
		out, _ := handler(impl, context.Background(), dec, nil)
		rsp := protojson.Format(out.(proto.Message))
		w.Write([]byte(rsp))
	} else {
		wrapper.inner.ServeHTTP(w, r)
	}
}
