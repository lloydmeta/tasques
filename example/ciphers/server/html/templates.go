package html

var (
	Index = `<!doctype html>
<html lang="en">
<head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

    <!-- Bootstrap CSS -->
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css"
          integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">

    <title>Messages and their ciphers: Leave a note</title>
</head>
<body>


<div class="jumbotron d-flex align-items-center">
    <div class="container">
        <a href="/"><h1>Messages and Ciphers</h1></a>
    </div>
</div>

<div class="d-flex align-items-center">
    <div class="container">
        <form action="/_create" method="post">
            <div class="form-group">
                <label for="Plain">Plain text</label>
                <input type="text" class="form-control" id="Plain" name="Plain" aria-describedby="plainHelp"
                       value="
{{if .Created}}
{{ .Created.Plain }}
{{else}}
ssssh it's a secret
{{end}}
">
                <small id="plainHelp" class="form-text text-muted">Just write anything you want</small>
            </div>
            <button type="submit" class="btn btn-primary">Submit</button>
        </form>
    </div>
</div>

<br/>
<br/>

<div class="d-flex align-items-center">
    <div class="container">
        <table class="table">
            <thead>
            <tr>
                <th scope="col">Plain</th>
                <th scope="col">Rot 1</th>
                <th scope="col">Rot 13</th>
                <th scope="col">Base64</th>
                <th scope="col">MD5</th>
                <th scope="col">Created at</th>
                <th scope="col">Modified at</th>
            </tr>
            </thead>
            <tbody>
            {{range .List}}
            <tr>
                <th scope="row">{{.Plain}}</th>

                <td>{{.Rot1}}</td>
                <td>{{.Rot13}}</td>
                <td>{{.Base64}}</td>
                <td>{{.Md5}}</td>
                <td>{{.CreatedAt}}</td>
                <td>{{.ModifiedAt}}</td>
            </tr>
            {{end}}
            </tbody>
        </table>
    </div>
</div>

<!-- Optional JavaScript -->
<!-- jQuery first, then Popper.js, then Bootstrap JS -->
<script src="https://code.jquery.com/jquery-3.2.1.slim.min.js"
        integrity="sha384-KJ3o2DKtIkvYIK3UENzmM7KCkRr/rE9/Qpg6aAZGJwFDMVNA/GpGFF93hXpG5KkN"
        crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.12.9/umd/popper.min.js"
        integrity="sha384-ApNbgh9B+Y1QKtv3Rn7W3mgPxhU9K/ScQsAP7hUibX39j7fakFPskvXusvfa0b4Q"
        crossorigin="anonymous"></script>
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/js/bootstrap.min.js"
        integrity="sha384-JZR6Spejh4U02d8jOt6vLEHfe/JQGiRRSQQxSfFWpi1MquVdAyjUar5+76PVCmYl"
        crossorigin="anonymous"></script>
</body>
</html>`
	Error = `<!doctype html>
<html lang="en">
<head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

    <!-- Bootstrap CSS -->
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css"
          integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">

    <title>Messages and their ciphers: Something went wrong</title>
</head>
<body>

<div class="jumbotron d-flex align-items-center">
    <div class="container">
        <h1>Error: {{ .Error }}</h1>
    </div>
</div>

<!-- Optional JavaScript -->
<!-- jQuery first, then Popper.js, then Bootstrap JS -->
<script src="https://code.jquery.com/jquery-3.2.1.slim.min.js"
        integrity="sha384-KJ3o2DKtIkvYIK3UENzmM7KCkRr/rE9/Qpg6aAZGJwFDMVNA/GpGFF93hXpG5KkN"
        crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.12.9/umd/popper.min.js"
        integrity="sha384-ApNbgh9B+Y1QKtv3Rn7W3mgPxhU9K/ScQsAP7hUibX39j7fakFPskvXusvfa0b4Q"
        crossorigin="anonymous"></script>
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/js/bootstrap.min.js"
        integrity="sha384-JZR6Spejh4U02d8jOt6vLEHfe/JQGiRRSQQxSfFWpi1MquVdAyjUar5+76PVCmYl"
        crossorigin="anonymous"></script>
</body>
</html>`
)
