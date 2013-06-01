package plugins

// I hate this, but I'm creating strings of the templates to avoid having to
// track where templates reside.

var factoidIndex string = `
<!DOCTYPE html>
<html>
<head>
	<title>Factoids</title>
	<link rel="stylesheet" href="http://yui.yahooapis.com/pure/0.1.0/pure-min.css">
</head>
	<div>
		<form action="/factoid/req" method="POST" class="pure-form">
			<fieldset>
				<legend>Search for a factoid</legend>
				<input type="text" name="entry" placeholder="trigger" value="{{.Search}}" />
				<button type="submit" class="pure-button notice">Find</button>
			</fieldset>
		</form>
	</div>

	<div>
		<style scoped>

	        .pure-button-success,
	        .pure-button-error,
	        .pure-button-warning,
	        .pure-button-secondary {
	            color: white;
	            border-radius: 4px;
	            text-shadow: 0 1px 1px rgba(0, 0, 0, 0.2);
	            padding: 2px;
	        }

	        .pure-button-success {
	            background: rgb(76, 201, 71); /* this is a green */
	        }

	        .pure-button-error {
	            background: rgb(202, 60, 60); /* this is a maroon */
	        }

	        .pure-button-warning {
	            background: orange;
	        }

	        .pure-button-secondary {
	            background: rgb(95, 198, 218); /* this is a light blue */
	        }

	    </style>

		{{if .Error}}
		<span id="error" class="pure-button-error">{{.Error}}</span>
		{{end}}

		{{if .Count}}
		<span id="count" class="pure-button-success">Found {{.Count}} entries.</span>
		{{end}}
	</div>

	{{if .Entries}}
	<div style="padding-top: 1em;">
		<table class="pure-table">
			<thead>
				<tr>
					<th>Trigger</th>
					<th>Full Text</th>
					<th>Author</th>
					<th># Hits</th>
				</tr>
			</thead>

			<tbody>
				{{range .Entries}}
				<tr>
					<td>{{.Trigger}}</td>
					<td>{{.FullText}}</td>
					<td>{{.CreatedBy}}</td>
					<td>{{.AccessCount}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
	{{end}}

</html>
`
