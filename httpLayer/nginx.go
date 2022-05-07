package httpLayer

import (
	"strings"
	"time"
)

const (
	nginx_timeFormatStr = "02 Jan 2006 15:04:05 MST"

	// real nginx response, echo xx | nc 127.0.0.1 80 > response
	Err400response_nginx = "HTTP/1.1 400 Bad Request\r\nServer: nginx/1.21.5\r\nDate: Sat, 02 Jan 2006 15:04:05 MST\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n<html>\r\n<head><title>400 Bad Request</title></head>\r\n<body>\r\n<center><h1>400 Bad Request</h1></center>\r\n<hr><center>nginx/1.21.5</center>\r\n</body>\r\n</html>\r\n"

	// real nginx response, curl -iv --raw 127.0.0.1/not_exist_path > response
	Err404response_nginx = "HTTP/1.1 404 Not Found\r\nServer: nginx/1.21.5\r\nDate: Sat, 02 Jan 2006 15:04:05 MST\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 19\r\nConnection: keep-alive\r\nCache-Control: no-cache, no-store, no-transform, must-revalidate, private, max-age=0\r\nExpires: Thu, 01 Jan 1970 08:00:00 AWST\r\nPragma: no-cache\r\nVary: Origin\r\nX-Content-Type-Options: nosniff\r\n\r\n404 page not found"

	Err403response_nginx = `HTTP/1.1 403 Forbidden\r\nServer: nginx/1.14.2\r\nDate: Sat, 07 May 2022 07:03:47 GMT\r\nContent-Type: text/html\r\nContent-Length: 169\r\nConnection: keep-alive\r\n\r\n<html>\r\n<head><title>403 Forbidden</title></head>\r\n<body bgcolor="white">\r\n<center><h1>403 Forbidden</h1></center>\r\n<hr><center>nginx/1.14.2</center>\r\n</body>\r\n</html>\r\n`
)

var nginxTimezone = time.FixedZone("GMT", 0)

//Get real a 400 response that looks like it comes from nginx.
func GetReal400Response() string {
	return GetRealResponse(Err400response_nginx)
}

//Get real a 403 response that looks like it comes from nginx.
func GetReal403Response() string {
	return GetRealResponse(Err403response_nginx)
}

//Get real a 404 response that looks like it comes from nginx.
func GetReal404Response() string {
	return GetRealResponse(Err404response_nginx)
}

//Get real a response that looks like it comes from nginx.
func GetRealResponse(template string) string {
	t := time.Now().UTC().In(nginxTimezone)

	tStr := t.Format(nginx_timeFormatStr)
	str := strings.Replace(template, nginx_timeFormatStr, tStr, 1)
	str = strings.Replace(str, "Sat", t.Weekday().String()[:3], 1)

	return str
}