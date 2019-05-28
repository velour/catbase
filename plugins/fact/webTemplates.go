// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package fact

// I hate this, but I'm creating strings of the templates to avoid having to
// track where templates reside.

// 2016-01-15 Later note, why are these in plugins and the server is in bot?

var factoidIndex = `
<!DOCTYPE html>
<html>
<head>
	<title>Factoids</title>
	<link rel="stylesheet" href="https://unpkg.com/purecss@1.0.0/build/pure-min.css" integrity="sha384-nn4HPE8lTHyVtfCBi5yW9d20FjT8BJwUXyWZT9InLYax14RDjBj46LmSztkmNP9w" crossorigin="anonymous">

	<!-- DataTables CSS -->
	<link rel="stylesheet" type="text/css" href="https://ajax.aspnetcdn.com/ajax/jquery.dataTables/1.9.4/css/jquery.dataTables.css">
	 
	<!-- jQuery -->
	<script type="text/javascript" charset="utf8" src="https://ajax.aspnetcdn.com/ajax/jQuery/jquery-1.8.2.min.js"></script>
	 
	<!-- DataTables -->
	<script type="text/javascript" charset="utf8" src="https://ajax.aspnetcdn.com/ajax/jquery.dataTables/1.9.4/jquery.dataTables.min.js"></script>

</head>
<body>
	<div>
		<form action="/factoid" method="GET" class="pure-form">
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
		<table class="pure-table" id="factTable">
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
					<td>{{linkify .Fact}}</td>
					<td>{{linkify .Tidbit}}</td>
					<td>{{linkify .Owner}}</td>
					<td>{{.Count}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
	{{end}}

	<script>
	$(document).ready(function(){
		$('#factTable').dataTable({
			"bPaginate": false
		});
	});
	</script>
</body>

</html>
`
