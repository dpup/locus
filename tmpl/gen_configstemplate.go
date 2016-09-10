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
padding: 2rem 0.2rem 0.2rem;
border-bottom: 0.3rem solid rgb(50, 50, 50);
font-weight: bold;
font-size: 1.2rem;
}

td span {
display: inline-block;
width: 7rem;
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
<tr>
<td colspan="2">Metrics</td>
</tr>
<tr>
<td>active connections:</td>
<td>{{.Connections.Count}}</td>
</tr>
<tr>
<td>requests:</td>
<td>
<span>count:</span> {{.Requests.Count}}<br>
<span>1-min rate:</span> {{.Requests.Rate1 | printf "%.2f"}}<br>
<span>5-min rate:</span> {{.Requests.Rate5 | printf "%.2f"}}<br>
<span>15-min rate:</span> {{.Requests.Rate15 | printf "%.2f"}}<br>
<span>mean rate:</span> {{.Requests.RateMean | printf "%.2f"}}
</td>
</tr>
<tr>
<td>5xx errors:</td>
<td>
<span>count:</span> {{.Errors.Count}}<br>
<span>1-min rate:</span> {{.Errors.Rate1 | printf "%.2f"}}<br>
<span>5-min rate:</span> {{.Errors.Rate5 | printf "%.2f"}}<br>
<span>15-min rate:</span> {{.Errors.Rate15 | printf "%.2f"}}
</td>
</tr>
<tr>
<td>latency:</td>
<td>
<span>min:</span> {{.Latency.Min}}<br>
<span>max:</span> {{.Latency.Max}}<br>
<span>avg:</span> {{.Latency.Mean | printf "%.0f"}}
</td>
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
{{range $i, $v := .UpstreamProvider.All}}
<tr>
<td>upstream #{{$i}}:</td>
<td>{{$v}}</td>
</tr>
{{end}}
{{range $k, $v :=.UpstreamProvider.DebugInfo}}
<tr>
<td>{{$k}}:</td>
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

