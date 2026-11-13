package main

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// listServices writes one line per service, or per method of one service.
func listServices(w io.Writer, service string) error {
	if service == "" {
		for _, sd := range services() {
			fmt.Fprintf(w, "%s\n", sd.Name())
		}
		return nil
	}
	sd, err := findService(service)
	if err != nil {
		return err
	}
	for i := 0; i < sd.Methods().Len(); i++ {
		fmt.Fprintln(w, methodSignature(sd.Methods().Get(i)))
	}
	return nil
}

// listAllMethods writes every method of every service.
func listAllMethods(w io.Writer) {
	for _, sd := range services() {
		for i := 0; i < sd.Methods().Len(); i++ {
			fmt.Fprintln(w, methodSignature(sd.Methods().Get(i)))
		}
	}
}

// methodSignature is e.g.
// "UserService/GetDevices(GetDevicesRequest) returns (stream Device) [auth: user]".
func methodSignature(md protoreflect.MethodDescriptor) string {
	in, out := string(md.Input().Name()), string(md.Output().Name())
	if md.IsStreamingClient() {
		in = "stream " + in
	}
	if md.IsStreamingServer() {
		out = "stream " + out
	}
	return fmt.Sprintf("%s(%s) returns (%s) [auth: %s]", methodName(md), in, out, methodAccess(md))
}

// describe writes a method's signature and the messages it uses, or a single
// message or enum, in proto-like syntax.
func describe(w io.Writer, name string) error {
	d := &describer{w: w, seen: map[protoreflect.FullName]bool{}}

	if md, err := findMethod(name); err == nil {
		fmt.Fprintf(w, "rpc %s\n", methodSignature(md))
		d.message(md.Input())
		d.message(md.Output())
		d.flush()
		return nil
	}

	full := protoreflect.FullName(name)
	if !strings.Contains(name, ".") {
		full = apiPackage + "." + full
	}
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(full)
	if err != nil {
		return fmt.Errorf("no method, message or enum named %q", name)
	}
	switch desc := desc.(type) {
	case protoreflect.MessageDescriptor:
		d.message(desc)
	case protoreflect.EnumDescriptor:
		d.enum(desc, "")
	case protoreflect.ServiceDescriptor:
		return listServices(w, string(desc.Name()))
	default:
		return fmt.Errorf("%q is not a method, message or enum", name)
	}
	d.flush()
	return nil
}

type describer struct {
	w     io.Writer
	seen  map[protoreflect.FullName]bool
	queue []protoreflect.MessageDescriptor
	enums []protoreflect.EnumDescriptor
}

// message queues md; flush writes queued messages and the messages they
// reference, each once.
func (d *describer) message(md protoreflect.MessageDescriptor) {
	if d.seen[md.FullName()] {
		return
	}
	d.seen[md.FullName()] = true
	d.queue = append(d.queue, md)
}

func (d *describer) flush() {
	for len(d.queue) > 0 {
		md := d.queue[0]
		d.queue = d.queue[1:]
		d.writeMessage(md)
	}
	for _, ed := range d.enums {
		fmt.Fprintln(d.w)
		d.enum(ed, "")
	}
	d.enums = nil
}

func (d *describer) writeMessage(md protoreflect.MessageDescriptor) {
	fmt.Fprintf(d.w, "\nmessage %s {\n", d.typeName(md.FullName()))
	for i := 0; i < md.Enums().Len(); i++ {
		d.enum(md.Enums().Get(i), "  ")
	}
	written := map[protoreflect.Name]bool{}
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if written[fd.Name()] {
			continue
		}
		if od := fd.ContainingOneof(); od != nil && !od.IsSynthetic() {
			fmt.Fprintf(d.w, "  oneof %s {\n", od.Name())
			for j := 0; j < od.Fields().Len(); j++ {
				of := od.Fields().Get(j)
				written[of.Name()] = true
				d.field(of, "    ")
			}
			fmt.Fprintf(d.w, "  }\n")
			continue
		}
		written[fd.Name()] = true
		d.field(fd, "  ")
	}
	fmt.Fprintf(d.w, "}\n")
}

func (d *describer) field(fd protoreflect.FieldDescriptor, indent string) {
	label := ""
	switch {
	case fd.IsMap():
	case fd.IsList():
		label = "repeated "
	case fd.HasOptionalKeyword():
		label = "optional "
	}
	fmt.Fprintf(d.w, "%s%s%s %s = %d;\n", indent, label, d.fieldType(fd), fd.Name(), fd.Number())
}

func (d *describer) fieldType(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return fmt.Sprintf("map<%s, %s>", d.fieldType(fd.MapKey()), d.fieldType(fd.MapValue()))
	}
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		d.message(fd.Message())
		return d.typeName(fd.Message().FullName())
	case protoreflect.EnumKind:
		ed := fd.Enum()
		// Nested enums are written inside their message; top-level ones
		// are written once after it.
		if _, ok := ed.Parent().(protoreflect.FileDescriptor); ok && !d.seen[ed.FullName()] {
			d.seen[ed.FullName()] = true
			d.enums = append(d.enums, ed)
		}
		return d.typeName(ed.FullName())
	}
	return fd.Kind().String()
}

func (d *describer) enum(ed protoreflect.EnumDescriptor, indent string) {
	fmt.Fprintf(d.w, "%senum %s {\n", indent, ed.Name())
	for i := 0; i < ed.Values().Len(); i++ {
		v := ed.Values().Get(i)
		fmt.Fprintf(d.w, "%s  %s = %d;\n", indent, v.Name(), v.Number())
	}
	fmt.Fprintf(d.w, "%s}\n", indent)
}

// typeName strips the API package from a full name.
func (d *describer) typeName(full protoreflect.FullName) string {
	return strings.TrimPrefix(string(full), apiPackage+".")
}
