package cmd

import (
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/iij/dagtools/env"
)

// Command defines methods for extension point
type Command interface {
	// Description is a command description
	Description() string
	// Usage returns a command usage
	Usage() string
	// Init a command
	Init(env *env.Environment) error
	// Run a command
	Run(args []string) error
}

var registry = struct {
	sync.Mutex
	extpoints map[string]*extensionPoint
}{
	extpoints: make(map[string]*extensionPoint),
}

type extensionPoint struct {
	sync.Mutex
	iface      reflect.Type
	components map[string]interface{}
}

func newExtensionPoint(iface interface{}) *extensionPoint {
	ep := &extensionPoint{
		iface:      reflect.TypeOf(iface).Elem(),
		components: make(map[string]interface{}),
	}
	registry.Lock()
	defer registry.Unlock()
	registry.extpoints[ep.iface.Name()] = ep
	return ep
}

func (ep *extensionPoint) lookup(name string) (ext interface{}, ok bool) {
	ep.Lock()
	defer ep.Unlock()
	ext, ok = ep.components[name]
	return
}

func (ep *extensionPoint) all() map[string]interface{} {
	ep.Lock()
	defer ep.Unlock()
	all := make(map[string]interface{})
	for k, v := range ep.components {
		all[k] = v
	}
	return all
}

func (ep *extensionPoint) register(component interface{}, name string) bool {
	ep.Lock()
	defer ep.Unlock()
	if name == "" {
		name = reflect.TypeOf(component).Elem().Name()
	}
	_, exists := ep.components[name]
	if exists {
		return false
	}
	ep.components[name] = component
	return true
}

func (ep *extensionPoint) unregister(name string) bool {
	ep.Lock()
	defer ep.Unlock()
	_, exists := ep.components[name]
	if !exists {
		return false
	}
	delete(ep.components, name)
	return true
}

func implements(component interface{}) []string {
	var ifaces []string
	for name, ep := range registry.extpoints {
		if reflect.TypeOf(component).Implements(ep.iface) {
			ifaces = append(ifaces, name)
		}
	}
	return ifaces
}

func Register(component interface{}, name string) []string {
	registry.Lock()
	defer registry.Unlock()
	var ifaces []string
	for _, iface := range implements(component) {
		if ok := registry.extpoints[iface].register(component, name); ok {
			ifaces = append(ifaces, iface)
		}
	}
	return ifaces
}

// Command

var Commands = &commandExt{
	newExtensionPoint(new(Command)),
}

type commandExt struct {
	*extensionPoint
}

func (ep *commandExt) Unregister(name string) bool {
	return ep.unregister(name)
}

func (ep *commandExt) Register(component Command, name string) bool {
	return ep.register(component, name)
}

func (ep *commandExt) Lookup(name string) (Command, bool) {
	ext, ok := ep.lookup(name)
	return ext.(Command), ok
}

func (ep *commandExt) All() map[string]Command {
	all := make(map[string]Command)
	for k, v := range ep.all() {
		all[k] = v.(Command)
	}
	return all
}

// Run dagtools
func Run(e *env.Environment, cmdName string, cmdArgs []string) int {
	e.Version = env.Version
	cmds := Commands.All()
	if cmds[cmdName] == nil {
		fmt.Fprintf(os.Stderr, "[Error] command not found: %q \n", cmdName)
		return 1
	}
	_cmd, _ := Commands.Lookup(cmdName)
	_cmd.Init(e)
	e.Logger.Printf("Starting %q command ..., args: %s", cmdName, cmdArgs)
	err := _cmd.Run(cmdArgs)
	if err != nil {
		if err == ErrArgument {
			fmt.Fprintf(os.Stderr, "[Error] illegal argument: %v\n", cmdArgs)
			fmt.Fprintln(os.Stderr, _cmd.Usage())
		} else {
			fmt.Fprintln(os.Stderr, "[Error]", err)
		}
		return 1
	}
	e.Logger.Printf("%q command finished. (elapsed time: %d ms)", cmdName, e.GetElapsedTimeMs())
	return 0
}
