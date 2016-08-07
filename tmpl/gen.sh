#!/bin/bash

name="$1"
classname="$(tr '[:lower:]' '[:upper:]' <<< ${name:0:1})${name:1}"
filename="gen_${name}template.go"

function write() {
  echo "$1" >> $filename
}

> $filename

write "/*"
write " * CODE GENERATED AUTOMATICALLY"
write " * THIS FILE SHOULD NOT BE EDITED BY HAND"
write " * Run 'go generate tmpl.go'"
write " */"
write "package tmpl"
write
write "// ${classname} is a generated constant that inlines ${name}.html"
write "var ${classname}Template = mustParse(\"${name}\", \`"

while read line; do
    write "$line"
done < ${name}.html

write "\`)"
write
