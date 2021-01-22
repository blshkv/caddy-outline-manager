package outline

var login = `<!DOCTYPE html>
<html lang="en">

<head>
  <meta charset="UTF-8">
  <title>Outlihe Manager</title>

  <style type="text/css">
    html {
      width: 100%;
      height: 100%;
      overflow: hidden;
      font-style: sans-serif;
    }

    body {
      width: 100%;
      height: 100%;
      font-family: 'Open Sans', sans-serif;
      margin: 0;
      background-color: #4A374A;
    }

    #login {
      position: absolute;
      top: 50%;
      left: 50%;
      margin: -150px 0 0 -150px;
      width: 300px;
      height: 300px;
    }

    #login h1 {
      color: #fff;
      text-shadow: 0 0 10px;
      letter-spacing: 1px;
      text-align: center;
    }

    h1 {
      font-size: 2em;
      margin: 0.67em 0;
    }

    input {
      width: 278px;
      height: 18px;
      margin-bottom: 10px;
      outline: none;
      padding: 10px;
      font-size: 13px;
      color: #fff;
      text-shadow: 1px 1px 1px;
      border-top: 1px solid #312E3D;
      border-left: 1px solid #312E3D;
      border-right: 1px solid #312E3D;
      border-bottom: 1px solid #56536A;
      border-radius: 4px;
      background-color: #2D2D3F;
    }

    .but {
      width: 300px;
      min-height: 20px;
      display: block;
      background-color: #4a77d4;
      border: 1px solid #3762bc;
      color: #fff;
      padding: 9px 14px;
      font-size: 15px;
      line-height: normal;
      border-radius: 5px;
      margin: 0;
    }
  </style>
</head>

<body onload = "JavaScript:on_load();">
  <div id="login">
    <h1>Outline Manager</h1>
    <input type="text" id="username" required="required" placeholder="Username" name="u"></input>
    <input type="password" id="password" required="required" placeholder="Password" name="p" onkeydown="if(event.keyCode==13){login();return false}"></input>
    <button class="but" type="button" onclick="login();">Login</button>
  </div>

  <script>
    function getCookie(cname) {
      var name = cname + "=";
      var decodedCookie = decodeURIComponent(document.cookie);
      var ca = decodedCookie.split(';');
      for(var i = 0; i < ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' ') {
          c = c.substring(1);
        }
        if (c.indexOf(name) == 0) {
          return c.substring(name.length, c.length);
        }
      }
      return "";
    }

    function setCookie(cname, cvalue, exdays) {
      var d = new Date();
      d.setTime(d.getTime() + (exdays*24*60*60*1000));
      var expires = "expires="+ d.toUTCString();
      document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/;SameSite=Lax";
    }

    function on_load() {
      var outline = getCookie("outline");
      if (outline != "") {
        console.log(outline);
        location.replace("/outline/manager");
      }
    }

    function login() {
      var user = document.getElementById("username").value;
      var pass = document.getElementById("password").value;

      var xmlHttp = new XMLHttpRequest();
      xmlHttp.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
          setCookie("outline", "manager", 30);
          location.replace("/outline/manager");
        } else {
        }
      }
      var url = document.URL + "/login?user=" + user + "&pass=" + pass
      xmlHttp.open("POST", url, false);
      xmlHttp.send(null);
    }
  </script>

</body>

</html>`
