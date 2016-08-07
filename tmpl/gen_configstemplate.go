/*
 * CODE GENERATED AUTOMATICALLY
 * THIS FILE SHOULD NOT BE EDITED BY HAND
 * Run 'go generate tmpl.go'
 */
package tmpl

// Configs is a generated constant that inlines configs.html
var ConfigsTemplate = mustParse("configs", `
<html>
<head>
<title>Locus Configs</title>
<style>
* {
font-family: -apple-system, ".SFNSText-Regular", "San Francisco", "Roboto", "Segoe UI", "Helvetica Neue", "Lucida Grande", sans-serif;
color: #333;
}

table {
border-collapse: collapse;
width: 100%;
max-width: 1000px;
margin: auto;
}

td {
padding: 5px;
vertical-align: top;
border-bottom: 1px solid #ccc;
font-size: 15px;
}

td:first-child {
width: 200px;
}

td[colspan="2"] {
padding-top: 20px;
border-bottom: 1px solid #666;
font-weight: bold;
font-size: 20px;
}
</style>
</head>
<body>
<table>
<tr>
<td colspan="2">Globals</td>
</tr>
<tr>
<td>local port:</td>
<td>{{.Port}}</td>
</tr>
<tr>
<td>read timeout:</td>
<td>{{.ReadTimeout}}</td>
</tr>
<tr>
<td>write timeout:</td>
<td>{{.WriteTimeout}}</td>
</tr>
<tr>
<td>verbose logging:</td>
<td>{{.VerboseLogging}}</td>
</tr>
{{range .Configs}}
<tr>
<td colspan="2">Site: {{.Name}}</td>
</tr>
<tr>
<td>binding:</td>
<td>{{.RequestMatcher}}</td>
</tr>
<tr>
<td>upstream:</td>
<td>{{range .UpstreamProvider.All}}
{{.}}<br>
{{end}}
</td>
</tr>
{{range $k, $v :=.UpstreamProvider.DebugInfo}}
<tr>
<td>upstream {{$k}}:</td>
<td>{{$v}}</td>
</tr>
{{end}}
{{end}}
</table>
</body>
</html>
`)
