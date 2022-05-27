package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/huyle49/shorten-url/database"
	"github.com/huyle49/shorten-url/helpers"
)

type request struct {
	URL											string 							`json:"url"`
	CustomShort 						string 							`json:"short"`
	Expire 									time.Duration 			`json:"expire"`
}

type response struct {
	URL 										string 							`json:"url"`
	CustomShort 						string 							`json:"short"`
	Expire 									time.Duration 			`json:"expire"`
	XRateRemaining 					int     						`json:"rate_limit"`	 
	XRateLimitReset 				time.Duration				`json:"rate_limit_reset"`
}

func ShortenUrl(c *fiber.Ctx) error {
	 
	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Cannot parse JSON"})
	}

	// implement rate limiting
	r2 := database.CreateClient(1)
	defer r2.Close()
	val, err := r2.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil{
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30000*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			// limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			val = "10"
		}
	}

	// check if the input if an actual URL

	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Invalid URL"})
	}

	// check for domain error
	if !helpers.RemoveDomainError(body.URL){
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error":"Domain err"})
	}

	//enforce https,ssl
	body.URL = helpers.EnforceHTTP(body.URL)
	var id string
	if body.CustomShort == ""{
		id = uuid.New().String()[:6]
	}else{
		id = body.CustomShort
	}

	r:=database.CreateClient(0)
	defer r.Close()

	val,_ = r.Get(database.Ctx, id).Result()
	if val != ""{
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":"URL custom short is already in use",
		})
	}

	if body.Expire == 0{
		body.Expire = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expire*3600*time.Second).Err()

	if err != nil{
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":"Unable to connect to server",
		})
	}

	resp := response{
		URL: body.URL,
		CustomShort:"",
		Expire:body.Expire,
		XRateRemaining: 10,
		XRateLimitReset: 30,
	}

	r2.Decr(database.Ctx, c.IP())


	val,_ = r2.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining,_ =  strconv.Atoi(val)

	ttl,_ := r2.TTL(database.Ctx, c.IP()).Result()

	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN")+ "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)

}