package routes

import (
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"
	"github.com/huyle49/shorten-url/database"
)

func ResolveURL (c *fiber.Ctx) error{

	url := c.Params("url")

	r := database.CreateClient(0)

	defer r.Close()
	value,err := r.Get(database.Ctx, url).Result()
	if err == redis.Nil{
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error":"Short not found in the database"})
	}else if err!= nil{
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error":"Cannot connect db"})
	}

	rInr := database.CreateClient(1)
	defer rInr.Close()
	_= rInr.Incr(database.Ctx, "counter")
	return c.Redirect(value, 301)

}