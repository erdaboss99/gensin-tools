package main

import (
	"fmt"
	"image/color"
	"strings"
)

// Colours sampled from the real game, where the scanner depends on them.
var (
	windowBackground = color.RGBA{42, 46, 60, 255}
	panelCream       = color.RGBA{236, 229, 216, 255}
	panelBanner      = color.RGBA{188, 105, 50, 255}
	chipDark         = color.RGBA{59, 66, 85, 255}
	iconOff          = color.RGBA{243, 239, 233, 255}
	bodyInk          = color.RGBA{73, 83, 102, 255}
	white            = color.RGBA{255, 255, 255, 255}
	mainStatLabel    = color.RGBA{192, 175, 169, 255}
	rarityGold       = color.RGBA{255, 204, 50, 255}
	elixirBand       = color.RGBA{220, 192, 255, 255}
	elixirInk        = color.RGBA{110, 70, 170, 255}
	medalGold        = color.RGBA{230, 190, 90, 255}
	earnedStar       = color.RGBA{254, 242, 212, 255}
	descriptionInk   = color.RGBA{130, 130, 130, 255}
	statusInk        = color.RGBA{150, 135, 110, 255}
	emptyMessageInk  = color.RGBA{150, 120, 90, 255}
)

const (
	// elixirShift is how far the elixir band pushes the panel's lower half down.
	elixirShift = 25
	// firstRowCenter and rowPitch place achievement rows, in reference pixels.
	firstRowCenter = 203
	rowPitch       = 83
	// headerGap is the extra space a category header adds above a row.
	headerGap = 27
	// descriptionLinePush is how far each extra description line moves the title up,
	// because a row's text is centred in its card.
	descriptionLinePush = 9

	emptySearchFile = "achievement_none_1280.png"
)

// inventoryFixture describes one artifact panel screenshot.
type inventoryFixture struct {
	file          string
	width, height int
	piece         string
	slot          string
	mainStat      string
	stars         int
	level         int
	substats      []string
	equipped      string
	locked        bool
	marked        bool
	elixir        bool
}

// achievementRow describes one row of search results.
type achievementRow struct {
	title  string
	stars  int
	status string
	// hasHeader leaves room for a category header above the row.
	hasHeader bool
	// descriptionLines is the number of description lines. Zero means one.
	descriptionLines int
}

// achievementFixture describes one achievement search screenshot.
type achievementFixture struct {
	file string
	rows []achievementRow
}

var inventoryFixtures = []inventoryFixture{
	{
		file: "artifact_circlet_1280.png", width: 1280, height: 720,
		piece: "Crown of the Befallen", slot: "Circlet of Logos", mainStat: "CRIT Rate", stars: 5, level: 20,
		substats: []string{"CRIT DMG+20.2%", "ATK+9.9%", "HP+299", "DEF+23"},
		equipped: "Hu Tao", locked: true, marked: true,
	},
	{
		file: "artifact_goblet_1920.png", width: 1920, height: 1080,
		piece: "Chalice of Your Blood and Sorrow", slot: "Goblet of Eonothem", mainStat: "Pyro DMG Bonus", stars: 5, level: 0,
		substats: []string{"CRIT Rate+3.9%", "Elemental Mastery+23", "Energy Recharge+5.8%", "ATK+5.8% (unactivated)"},
	},
	{
		file: "artifact_flower_4star_1280.png", width: 1280, height: 800,
		piece: "Gladiator's Nostalgia", slot: "Flower of Life", mainStat: "HP", stars: 4, level: 16,
		substats: []string{"ATK+16", "CRIT Rate+3.1%", "DEF+5.8%", "HP+4.2%"},
		equipped: "Kaedehara Kazuha", locked: true,
	},
	{
		file: "artifact_wrapped_name_1280.png", width: 1280, height: 720,
		piece: "Sharpness That Ceased Upon\nWondrous Creation", slot: "Plume of Death", mainStat: "ATK", stars: 5, level: 4,
		substats: []string{"CRIT Rate+3.1%", "CRIT DMG+6.2%", "HP+5.8%", "DEF+23"},
	},
	{
		file: "artifact_elixir_1920.png", width: 1920, height: 1080,
		piece: "Revelation's Toll", slot: "Sands of Eon", mainStat: "ATK", stars: 5, level: 20,
		substats: []string{"CRIT Rate+7.4%", "CRIT DMG+17.9%", "DEF+7.3%", "Elemental Mastery+44"},
		equipped: "Flins", locked: true, marked: true, elixir: true,
	},
	{
		file: "material_1280.png", width: 1280, height: 720,
		piece: "Sanctifying Essence", slot: "Artifact EXP", stars: 3,
	},
}

var achievementFixtures = []achievementFixture{
	{file: "achievement_single_1280.png", rows: []achievementRow{
		{title: "Zoo Tycoon", stars: 2, status: "2/100"},
	}},
	{file: "achievement_multiple_1280.png", rows: []achievementRow{
		{title: "Tales of Monstrous Madness", status: "0/1"},
		{title: "To Brave the Lightning's Glow", stars: 1, status: "Completed"},
		{title: "Brave the Lightning's Glow...", stars: 1, status: "Completed", hasHeader: true},
		{title: "Zoo Tycoon", stars: 3, status: "Completed", hasHeader: true},
	}},
	{file: "achievement_tall_row_1280.png", rows: []achievementRow{
		{title: "Urban Combat Specialist", status: "0/3", descriptionLines: 3},
	}},
}

// paintError keeps drawing code readable: the first text failure sticks, and later
// calls do nothing, so the sequence of drawing steps needs no error checks.
type paintError struct{ err error }

func (p *paintError) text(c *canvas, x, y float64, text string, size float64, fill color.RGBA, weight fontWeight) {
	if p.err == nil {
		p.err = c.text(x, y, text, size, fill, weight)
	}
}

func drawInventory(fonts fontSet, f inventoryFixture) *canvas {
	c := newCanvas(fonts, f.width, f.height, windowBackground)
	var paint paintError
	paint.text(c, -250, 23, "Artifacts 123/2100", 16, panelCream, weightRegular)
	c.rect(-408, 80, -81, -80, panelCream)
	c.rect(-408, 80, -81, 268, panelBanner)
	nameSize := 19.0
	if strings.Contains(f.piece, "\n") {
		nameSize = 13
	}
	paint.text(c, -391, 84, f.piece, nameSize, white, weightBold)
	paint.text(c, -391, 123, f.slot, 14, white, weightRegular)
	paint.text(c, -391, 182, f.mainStat, 14, mainStatLabel, weightRegular)
	for i := range f.stars {
		c.star(-381+float64(i)*23, 248, 9, rarityGold)
	}

	shift := 0.0
	if f.elixir {
		c.rect(-408, 269, -81, 297, elixirBand)
		paint.text(c, -370, 275, "Sanctifying Elixir Definition", 14, elixirInk, weightRegular)
		shift = elixirShift
	}

	c.rect(-391, 284+shift, -349, 307+shift, chipDark)
	paint.text(c, -386, 288+shift, fmt.Sprintf("+%d", f.level), 14, white, weightRegular)
	c.rect(-162, 280+shift, -134, 308+shift, toggleColor(f.locked))
	c.rect(-124, 280+shift, -96, 308+shift, toggleColor(f.marked))
	for i, line := range f.substats {
		paint.text(c, -378, 318+shift+float64(i)*25, line, 15, bodyInk, weightRegular)
	}
	if f.equipped != "" {
		paint.text(c, -272, -108, f.equipped, 15, bodyInk, weightRegular)
	}
	mustPaint(f.file, paint)
	return c
}

func toggleColor(on bool) color.RGBA {
	if on {
		return chipDark
	}
	return iconOff
}

func drawAchievements(fonts fontSet, f achievementFixture) *canvas {
	c := newCanvas(fonts, referenceWidth, 720, windowBackground)
	var paint paintError
	center := float64(firstRowCenter)
	for _, row := range f.rows {
		if row.hasHeader {
			center += headerGap
		}
		drawAchievementRow(c, &paint, row, center)
		center += rowPitch
	}
	mustPaint(f.file, paint)
	return c
}

func drawAchievementRow(c *canvas, paint *paintError, row achievementRow, center float64) {
	c.rect(492, center-37, 1212, center+41, panelCream)
	for _, inset := range []float64{0, 4} {
		c.ellipseOutline(504+inset, center-28+inset, 566-inset, center+28-inset, 3, medalGold)
	}
	for s := range row.stars {
		c.star(540+float64(s)*14-float64(row.stars-1)*7, center, 6, earnedStar)
	}
	lines := max(row.descriptionLines, 1)
	titleTop := center - 20 - float64(lines-1)*descriptionLinePush
	paint.text(c, 578, titleTop, row.title, 17, bodyInk, weightBold)
	for line := range lines {
		paint.text(c, 578, titleTop+28+float64(line)*17, "Do the thing that this achievement asks for.", 12, descriptionInk, weightRegular)
	}
	paint.text(c, 1125, center-8, row.status, 13, statusInk, weightRegular)
}

func drawEmptySearch(fonts fontSet) *canvas {
	c := newCanvas(fonts, referenceWidth, 720, windowBackground)
	var paint paintError
	paint.text(c, 578, 406, `No achievements related to "Tales of Monstrous Madness"`, 15, emptyMessageInk, weightRegular)
	mustPaint(emptySearchFile, paint)
	return c
}

// mustPaint stops the program on a font failure, which can only mean a broken build
// of the embedded fonts.
func mustPaint(file string, paint paintError) {
	if paint.err != nil {
		panic(fmt.Sprintf("%s: %v", file, paint.err))
	}
}
