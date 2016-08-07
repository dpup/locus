/*
 * CODE GENERATED AUTOMATICALLY
 * THIS FILE SHOULD NOT BE EDITED BY HAND
 * Run 'go generate tmpl.go'
 */
package tmpl

// Error is a generated constant that inlines error.html
var ErrorTemplate = mustParse("error", `
<html>
<head>
<title>{{.Status}} Error</title>
<style>
body {
display: flex;
background-color: rgb(228, 240, 245);
align-items: center;
}

h1 {
font-family: -apple-system, ".SFNSText-Regular", "San Francisco", "Roboto", "Segoe UI", "Helvetica Neue", "Lucida Grande", sans-serif;
font-size: 5rem;
margin: auto;
padding: 1rem;
border: 0.5rem solid rgb(255, 255, 255);
color: rgb(255, 255, 255);
}
</style>
</head>
<body>
<h1>
{{.Status}}
{{if eq .Status 400}}Bad Request
{{else if eq .Status 401}}Unauthorized
{{else if eq .Status 402}}Payment Required
{{else if eq .Status 403}}Forbidden
{{else if eq .Status 404}}Not Found
{{else if eq .Status 405}}Method Not Allowed
{{else if eq .Status 406}}Not Acceptable
{{else if eq .Status 407}}Proxy Authentication Required
{{else if eq .Status 408}}Request Timeout
{{else if eq .Status 409}}Conflict
{{else if eq .Status 410}}Gone
{{else if eq .Status 413}}Payload To Large
{{else if eq .Status 414}}URI Too Long
{{else if eq .Status 431}}Request Header Fields Too Large
{{else if eq .Status 500}}Internal Server Error
{{else if eq .Status 501}}Not Implemented
{{else if eq .Status 502}}Bad Gateway
{{else if eq .Status 503}}Service Unavailable
{{else if eq .Status 504}}Gateway Timeout
{{else}}Error
{{end}}
</h1>
</body>
</html>
`)

