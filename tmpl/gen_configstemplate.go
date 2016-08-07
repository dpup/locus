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
}
body {
display: flex;
background-color: rgb(228, 240, 245);
align-items: center;
color: rgb(50, 50, 50);
}

table {
border-collapse: collapse;
width: 100%;
max-width: 1000px;
margin: auto;
}

td {
color: rgb(50, 50, 50);
padding: 0.5rem 0.2rem;
vertical-align: top;
border-bottom: 0.1rem solid rgb(50, 50, 50);
font-size: 1rem;
}

td:first-child {
width: 200px;
}

td[colspan="2"] {
padding: 2rem 0.5rem 0.2rem;
border-bottom: 0.3rem solid rgb(50, 50, 50);
font-weight: bold;
font-size: 1.2rem;
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
<td colspan="2">
{{if .Redirect}}
Redirect:
{{else}}
Site:
{{end}}
{{.Name}}
</td>
</tr>
<tr>
<td>binding:</td>
<td>{{.Matcher}}</td>
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
{{if .Redirect}}
<tr>
<td>redirect</td>
<td>{{.Redirect}}</td>
</tr>
{{end}}
{{end}}
</table>
</body>
</html>
`)

