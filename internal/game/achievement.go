package game

// AchievementID identifies one achievement stage.
type AchievementID int

// CategoryID identifies an achievement category.
type CategoryID int

// CompletedAchievement is one completed achievement stage.
type CompletedAchievement struct {
	ID       AchievementID `json:"id"`
	Category CategoryID    `json:"category"`
}
