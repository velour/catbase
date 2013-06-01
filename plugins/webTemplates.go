package plugins

// I hate this, but I'm creating strings of the templates to avoid having to
// track where templates reside.

var factoidIndex string = `
<!DOCTYPE html>
<html>
<head>
<title>Factoids</title>
</head>

	{{if .Error}}
	<div id="error">{{.Error}}</div>
	{{end}}

	<div>
		<form action="/factoid/req" method="POST">
			<input type="text" name="entry" /> <input type="submit" value="Find" />
		</form>
	</div>

	{{ $entries := .Entries }}

	{{if .Count}}
	<div id="count">Found {{.Count}} entries.</div>
	{{end}}

	{{range $entries}}
		<div class="entry">
			{{.Trigger}} - {{.Action}}
		</div>
	{{end}}

</html>
`
