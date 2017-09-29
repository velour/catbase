package twitch

var page = `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Is {{.Name}} streaming?</title>

</head>

<body style="text-align: center; padding-top: 200px;">

<a style="font-weight: bold; font-size: 120pt;
font-family: Arial, sans-serif; text-decoration: none; color: black;"
title="{{.Status}}">{{.Status}}</a>

</body>
</html>
`
