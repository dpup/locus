#!/bin/bash

echo "/*" > tmpl_gen.go
echo " * CODE GENERATED AUTOMATICALLY" >> tmpl_gen.go
echo " * THIS FILE SHOULD NOT BE EDITED BY HAND" >> tmpl_gen.go
echo " * Run 'go generate tmpl.go'" >> tmpl_gen.go
echo " */" >> tmpl_gen.go
echo "package tmpl" >> tmpl_gen.go
echo >> tmpl_gen.go
echo "// DebugTemplate is a generated constant that inlines debug.html" >> tmpl_gen.go
echo "var DebugTemplate = mustParse(\"debug\", \`" >> tmpl_gen.go
cat debug.html >> tmpl_gen.go
echo "\`)" >> tmpl_gen.go
