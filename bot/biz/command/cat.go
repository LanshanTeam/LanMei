package command

import (
	"LanMei/bot/utils/file"
	"fmt"
	"math/rand"
	"time"
)

const (
	HTTP100 = "https://http.cat/100.jpg"
	HTTP101 = "https://http.cat/101.jpg"
	HTTP102 = "https://http.cat/102.jpg"
	HTTP103 = "https://http.cat/103.jpg"
	HTTP200 = "https://http.cat/200.jpg"
	HTTP201 = "https://http.cat/201.jpg"
	HTTP202 = "https://http.cat/202.jpg"
	HTTP203 = "https://http.cat/203.jpg"
	HTTP204 = "https://http.cat/204.jpg"
	HTTP205 = "https://http.cat/205.jpg"
	HTTP206 = "https://http.cat/206.jpg"
	HTTP207 = "https://http.cat/207.jpg"
	HTTP208 = "https://http.cat/208.jpg"
	HTTP214 = "https://http.cat/214.jpg"
	HTTP226 = "https://http.cat/226.jpg"

	HTTP300 = "https://http.cat/300.jpg"
	HTTP301 = "https://http.cat/301.jpg"
	HTTP302 = "https://http.cat/302.jpg"
	HTTP303 = "https://http.cat/303.jpg"
	HTTP304 = "https://http.cat/304.jpg"
	HTTP305 = "https://http.cat/305.jpg"
	HTTP307 = "https://http.cat/307.jpg"
	HTTP308 = "https://http.cat/308.jpg"

	HTTP400 = "https://http.cat/400.jpg"
	HTTP401 = "https://http.cat/401.jpg"
	HTTP402 = "https://http.cat/402.jpg"
	HTTP403 = "https://http.cat/403.jpg"
	HTTP404 = "https://http.cat/404.jpg"
	HTTP405 = "https://http.cat/405.jpg"
	HTTP406 = "https://http.cat/406.jpg"
	HTTP407 = "https://http.cat/407.jpg"
	HTTP408 = "https://http.cat/408.jpg"
	HTTP409 = "https://http.cat/409.jpg"
	HTTP410 = "https://http.cat/410.jpg"
	HTTP411 = "https://http.cat/411.jpg"
	HTTP412 = "https://http.cat/412.jpg"
	HTTP413 = "https://http.cat/413.jpg"
	HTTP414 = "https://http.cat/414.jpg"
	HTTP415 = "https://http.cat/415.jpg"
	HTTP416 = "https://http.cat/416.jpg"
	HTTP417 = "https://http.cat/417.jpg"
	HTTP418 = "https://http.cat/418.jpg"
	HTTP419 = "https://http.cat/419.jpg"
	HTTP420 = "https://http.cat/420.jpg"
	HTTP421 = "https://http.cat/421.jpg"
	HTTP422 = "https://http.cat/422.jpg"
	HTTP423 = "https://http.cat/423.jpg"
	HTTP424 = "https://http.cat/424.jpg"
	HTTP425 = "https://http.cat/425.jpg"
	HTTP426 = "https://http.cat/426.jpg"
	HTTP428 = "https://http.cat/428.jpg"
	HTTP429 = "https://http.cat/429.jpg"
	HTTP431 = "https://http.cat/431.jpg"
	HTTP444 = "https://http.cat/444.jpg"
	HTTP450 = "https://http.cat/450.jpg"
	HTTP451 = "https://http.cat/451.jpg"
	HTTP495 = "https://http.cat/495.jpg"
	HTTP496 = "https://http.cat/496.jpg"
	HTTP497 = "https://http.cat/497.jpg"
	HTTP498 = "https://http.cat/498.jpg"
	HTTP499 = "https://http.cat/499.jpg"

	HTTP500 = "https://http.cat/500.jpg"
	HTTP501 = "https://http.cat/501.jpg"
	HTTP502 = "https://http.cat/502.jpg"
	HTTP503 = "https://http.cat/503.jpg"
	HTTP504 = "https://http.cat/504.jpg"
	HTTP506 = "https://http.cat/506.jpg"
	HTTP507 = "https://http.cat/507.jpg"
	HTTP508 = "https://http.cat/508.jpg"
	HTTP509 = "https://http.cat/509.jpg"
	HTTP510 = "https://http.cat/510.jpg"
	HTTP511 = "https://http.cat/511.jpg"
	HTTP521 = "https://http.cat/521.jpg"
	HTTP522 = "https://http.cat/522.jpg"
	HTTP523 = "https://http.cat/523.jpg"
	HTTP525 = "https://http.cat/525.jpg"
	HTTP530 = "https://http.cat/530.jpg"
	HTTP599 = "https://http.cat/599.jpg"
)

var HTTPCatURLs = []string{
	HTTP100, HTTP101, HTTP102, HTTP103,
	HTTP200, HTTP201, HTTP202, HTTP203, HTTP204, HTTP205, HTTP206, HTTP207, HTTP208, HTTP214, HTTP226,
	HTTP300, HTTP301, HTTP302, HTTP303, HTTP304, HTTP305, HTTP307, HTTP308,
	HTTP400, HTTP401, HTTP402, HTTP403, HTTP404, HTTP405, HTTP406, HTTP407, HTTP408, HTTP409, HTTP410, HTTP411, HTTP412, HTTP413, HTTP414, HTTP415, HTTP416, HTTP417, HTTP418, HTTP419, HTTP420, HTTP421, HTTP422, HTTP423, HTTP424, HTTP425, HTTP426, HTTP428, HTTP429, HTTP431, HTTP444, HTTP450, HTTP451, HTTP495, HTTP496, HTTP497, HTTP498, HTTP499,
	HTTP500, HTTP501, HTTP502, HTTP503, HTTP504, HTTP506, HTTP507, HTTP508, HTTP509, HTTP510, HTTP511, HTTP521, HTTP522, HTTP523, HTTP525, HTTP530, HTTP599,
}

func GetHttpCat(input string, groupId string) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if input == "" {
		Select := r.Intn(len(HTTPCatURLs))
		return file.UploadPicAndStore(HTTPCatURLs[Select], groupId)
	} else {
		return file.UploadPicAndStore(fmt.Sprintf("https://http.cat/%s.jpg", input), groupId)
	}
}
