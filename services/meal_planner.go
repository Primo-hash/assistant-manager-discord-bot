package services

import (
	"assistant/DB"
	dataRequests "assistant/DataRequests"
	"assistant/utils"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func HandleRouteToMeals(subRoute string, flags map[string]string, uid string) ([]discordgo.MessageEmbed, error) {
	var mealEmbed = []discordgo.MessageEmbed{}

	switch subRoute {
	case utils.View:
		return createViewMessage(uid)
	case utils.Get, utils.Check:
		if len(flags) != 0 {
			return mealEmbed, nil
		} else {
			//Get from fridge
			recipes, err := getRecipeFromFridge(uid)
			if err != nil {
				return mealEmbed, err
			}
			return createRecipeMessages(recipes), nil
		}
	case utils.Add, utils.Set:
		if len(flags) != 0 {
			if ingredient, ok := flags[utils.Ingredient]; ok {
				err := addToFridge(ingredient, uid)
				if err != nil {
					return mealEmbed, err
				}
				var info = discordgo.MessageEmbed{}
				info.Title = "Added ingredient " + ingredient
				mealEmbed = append(mealEmbed, info)
				return mealEmbed, nil
			}
			return mealEmbed, errors.New("ingredient flag is required")
		} else {
			return mealEmbed, errors.New("flags are needed")
		}
	case utils.Delete, utils.Remove:
		if len(flags) != 0 {
			if ingredient, ok := flags[utils.Ingredient]; ok {
				err := removeFromFridge(ingredient, uid)
				if err != nil {
					return mealEmbed, err
				}
				var info = discordgo.MessageEmbed{}
				info.Title = "Removed ingredient " + ingredient
				mealEmbed = append(mealEmbed, info)
				return mealEmbed, nil
			}
			return mealEmbed, errors.New("ingredient flag is required")
		} else {
			return mealEmbed, errors.New("flags are needed")
		}
	default:
		return mealEmbed, errors.New("sub route not recognized")
	}
}

func getRecipeFromFridge(uid string) (utils.Recipe, error) {
	//Use a test fridge until we have an implementation of UserData
	fridge, err := retrieveFridgeIngredients(uid)
	if err != nil {
		return utils.Recipe{}, err
	}

	//Check if fridge is empty
	if len(fridge.Ingredients) == 0 {
		return utils.Recipe{}, errors.New("fridge is empty")
	}
	//Fridge is not empty
	var ingredientString string
	for _, ingredient := range fridge.Ingredients {
		ingredientString += ingredient + ","
	}
	number := "5"
	//Create url and recipe struct for holding data
	url := "https://api.spoonacular.com/recipes/findByIngredients?ingredients=" + ingredientString + "&number=" + number + "&apiKey=" + utils.MealKey
	var recipe utils.Recipe
	//Use GetAndDecode function to decode it into recipe struct
	requestError := dataRequests.GetAndDecodeURL(url, &recipe)
	//Check if there was any errors in fetching and decoding the url
	if requestError != nil {
		return utils.Recipe{}, err
	}
	return recipe, nil
}

func createRecipeMessages(recipes utils.Recipe) []discordgo.MessageEmbed {
	var messageArray []discordgo.MessageEmbed
	for _, recipe := range recipes {
		recipeMessage := discordgo.MessageEmbed{}
		recipeMessage.Title = recipe.Name
		recipeMessage.Image = &discordgo.MessageEmbedImage{URL: recipe.Image}

		var missedIngredients, usedIngredients string
		for _, ingredients := range recipe.MissedIngredients {
			missedIngredients += strings.Title(ingredients.IngredientName) + "\n"
		}
		//Embed missed ingredients
		fieldMissed := discordgo.MessageEmbedField{Name: "Missed ingredients: ", Value: missedIngredients}
		fields := []*discordgo.MessageEmbedField{&fieldMissed}
		if recipe.UsedIngredientsCount > 0 {
			for _, ingredients := range recipe.UsedIngredients {
				usedIngredients += strings.Title(ingredients.IngredientName)
			}
			fieldUsed := discordgo.MessageEmbedField{Name: "Used Ingredients: ", Value: usedIngredients}
			fields = append(fields, &fieldUsed)
		}

		// Create footer
		footer := discordgo.MessageEmbedFooter{Text: "Data provided by"}

		// Set footer and fields
		recipeMessage.Fields = fields
		recipeMessage.Footer = &footer
		messageArray = append(messageArray, recipeMessage)
	}
	return messageArray
}

// createTestFridge returns a fridge with some ingredients
func createTestFridge() utils.Fridge {
	var fridge utils.Fridge
	fridge.Ingredients = append(fridge.Ingredients, "Apple", "Milk", "Chicken", "Butter")

	return fridge
}

// retrieveFridge Retrieve fridge with its ingredients from the database
func retrieveFridge(uid string) (map[string]interface{}, error) {
	fmt.Println("Retrieve from fridge command")
	// Retrieve the fridge entry from the database
	fridge, err := DB.RetrieveFromDatabase("fridge", uid)
	if err != nil {
		return fridge, err
	}
	return fridge, nil
}

// retrieveFridgeIngredients Retrieves the ingredients in the format required for recipe searching
func retrieveFridgeIngredients(uid string) (utils.Fridge, error) {
	// Output variable
	var fridgeIngredients utils.Fridge
	// Retrieve the fridge from the database
	fridge, err := retrieveFridge(uid)
	if err != nil {
		return utils.Fridge{}, err
	}
	// Add the fridge ingredients to the fridgeIngredients
	for ingredient := range fridge {
		fridgeIngredients.Ingredients = append(fridgeIngredients.Ingredients, ingredient)
	}
	// Return the formatted ingredients
	return fridgeIngredients, nil
}

// addToFridge Adds an ingredient to the database
func addToFridge(ingredient string, uid string) error {
	fmt.Println("Add to fridge command")
	// Retrieve the fridge from the database
	fridge, err := retrieveFridge(uid)
	if err != nil {
		return err
	}
	// Add the ingredient to the fridge
	fridge[ingredient] = "1"
	// Send the updated fridge to the database
	DB.AddToDatabase("fridge", uid, fridge)

	return nil
}

// removeFromFridge Removes an ingredient from the database
func removeFromFridge(ingredient string, uid string) error {
	// Retrieve the fridge from the database
	fridge, err := retrieveFridge(uid)
	if err != nil {
		return err
	}
	// Remove the ingredient
	delete(fridge, ingredient)
	// Send the updated fridge to the database
	DB.AddToDatabase("fridge", uid, fridge)

	return nil
}

func createViewMessage(uid string) ([]discordgo.MessageEmbed, error) {
	fridge, err := retrieveFridgeIngredients(uid)
	if err != nil {
		return []discordgo.MessageEmbed{}, err
	}
	//fridge := createTestFridge()

	var messageList []discordgo.MessageEmbed
	var message discordgo.MessageEmbed

	//Create message
	message.Title = "Your Fridge"

	var ingredients string
	for _, ingredient := range fridge.Ingredients {
		ingredients += ingredient + "\n"
	}
	if len(fridge.Ingredients) < 1 {
		ingredients = "There are no ingredients stored in your fridge"
	}
	//Embed ingredients to message
	fridgeContent := discordgo.MessageEmbedField{Name: "Ingredients: ", Value: ingredients}
	fields := []*discordgo.MessageEmbedField{&fridgeContent}

	message.Fields = fields
	messageList = append(messageList, message)
	return []discordgo.MessageEmbed{}, nil
}
