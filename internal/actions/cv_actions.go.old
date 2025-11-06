package actions

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

// CV-based action primitives

// FindTemplate finds a template in the current screen
func (ab *ActionBuilder) FindTemplate(template cv.Template) *ActionBuilder {
	templatePath := buildTemplatePath(template.Name)
	step := Step{
		name: fmt.Sprintf("FindTemplate(%s)", buildTemplatePath(template.Name)),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: template.Threshold,
			}

			result, err := ab.bot.CV().FindTemplate(templatePath, config)
			if err != nil {
				return fmt.Errorf("failed to find template: %w", err)
			}

			if !result.Found {
				return fmt.Errorf("template not found")
			}

			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// WaitForTemplate waits for a template to appear
func (ab *ActionBuilder) WaitForTemplate(template cv.Template, timeout time.Duration) *ActionBuilder {
	templatePath := buildTemplatePath(template.Name)
	threshold := template.Threshold
	step := Step{
		name: fmt.Sprintf("WaitForTemplate(%s, %v)", templatePath, timeout),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			result, err := ab.bot.CV().WaitForTemplate(templatePath, config, timeout)
			if err != nil {
				return fmt.Errorf("template wait timeout: %w", err)
			}

			if !result.Found {
				return fmt.Errorf("template not found within timeout")
			}

			return nil
		},
		timeout: timeout,
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// FindAndClickTemplate finds a template and clicks it
func (ab *ActionBuilder) FindAndClickTemplate(template cv.Template, offsetX int, offsetY int) *ActionBuilder {
	templatePath := buildTemplatePath(template.Name)
	threshold := template.Threshold
	step := Step{
		name: fmt.Sprintf("FindAndClickTemplate(%s)", templatePath),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			result, err := ab.bot.CV().FindTemplate(templatePath, config)
			if err != nil {
				return fmt.Errorf("failed to find template: %w", err)
			}

			if !result.Found {
				return fmt.Errorf("template not found")
			}

			// Click at template location + offset
			clickX := result.Location.X + offsetX
			clickY := result.Location.Y + offsetY

			return ab.bot.ADB().Click(clickX, clickY)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// FindAndClickCenter finds a template and clicks its center
func (ab *ActionBuilder) FindAndClickCenter(template cv.Template) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("FindAndClickCenter(%s)", template.Name),
		execute: func() error {
			config := buildMatchConfig(template, cv.MatchMethodNCC)

			// Get template size
			img, err := loadTemplateImage(template.Name)
			if err != nil {
				return fmt.Errorf("failed to load template: %w", err)
			}

			result, err := ab.bot.CV().FindTemplate(template.Name, config)
			if err != nil {
				return fmt.Errorf("failed to find template: %w", err)
			}

			if !result.Found {
				return fmt.Errorf("template %s not found (confidence: %.2f, threshold: %.2f)",
					template.Name, result.Confidence, config.Threshold)
			}

			// Click at center of template
			bounds := img.Bounds()
			centerX := result.Location.X + bounds.Dx()/2
			centerY := result.Location.Y + bounds.Dy()/2
			fmt.Printf("Found %s at (%d,%d), clicking center (%d,%d), confidence: %.2f\n",
				template.Name, result.Location.X, result.Location.Y, centerX, centerY, result.Confidence)

			return ab.bot.ADB().Click(centerX, centerY)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// WaitForTemplateAndClick waits for template then clicks it
func (ab *ActionBuilder) WaitForTemplateAndClick(template cv.Template, timeout time.Duration, offsetX, offsetY int) *ActionBuilder {
	templatePath := buildTemplatePath(template.Name)
	threshold := template.Threshold
	step := Step{
		name: fmt.Sprintf("WaitForTemplateAndClick(%s)", templatePath),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			result, err := ab.bot.CV().WaitForTemplate(templatePath, config, timeout)
			if err != nil {
				return fmt.Errorf("template wait timeout: %w", err)
			}

			if !result.Found {
				return fmt.Errorf("template not found within timeout")
			}

			// Click at template location + offset
			clickX := result.Location.X + offsetX
			clickY := result.Location.Y + offsetY

			return ab.bot.ADB().Click(clickX, clickY)
		},
		timeout: timeout,
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// WaitForTemplateDisappear waits for a template to disappear (e.g., loading screen)
func (ab *ActionBuilder) WaitForTemplateDisappear(template cv.Template, timeout time.Duration) *ActionBuilder {
	templatePath := buildTemplatePath(template.Name)
	threshold := template.Threshold
	step := Step{
		name: fmt.Sprintf("WaitForTemplateDisappear(%s)", templatePath),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			start := time.Now()
			for {
				ab.bot.CV().InvalidateCache()
				result, err := ab.bot.CV().FindTemplate(templatePath, config)
				if err != nil {
					return fmt.Errorf("error checking template: %w", err)
				}

				// Template disappeared!
				if !result.Found {
					return nil
				}

				if time.Since(start) > timeout {
					return fmt.Errorf("template did not disappear within timeout")
				}

				time.Sleep(100 * time.Millisecond)
			}
		},
		timeout: timeout,
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// WaitForAnyTemplate waits for any of multiple templates to appear
func (ab *ActionBuilder) WaitForAnyTemplate(templatePaths []string, threshold float64, timeout time.Duration) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("WaitForAnyTemplate(%d templates)", len(templatePaths)),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			start := time.Now()
			for {
				ab.bot.CV().InvalidateCache()

				results, err := ab.bot.CV().FindMultipleTemplates(templatePaths, config)
				if err != nil {
					return fmt.Errorf("error checking templates: %w", err)
				}

				// Check if any found
				for _, result := range results {
					if result.Found {
						return nil // Found one!
					}
				}

				if time.Since(start) > timeout {
					return fmt.Errorf("no templates found within timeout")
				}

				time.Sleep(100 * time.Millisecond)
			}
		},
		timeout: timeout,
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// CheckColor verifies a pixel has expected color
func (ab *ActionBuilder) CheckColor(x, y int, expected color.Color, tolerance uint8) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("CheckColor(%d, %d)", x, y),
		execute: func() error {
			matches, err := ab.bot.CV().CheckColor(x, y, expected, tolerance)
			if err != nil {
				return fmt.Errorf("failed to check color: %w", err)
			}

			if !matches {
				return fmt.Errorf("color mismatch at (%d, %d)", x, y)
			}

			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// WaitForColor waits for a pixel to become a specific color
func (ab *ActionBuilder) WaitForColor(x, y int, expected color.Color, tolerance uint8, timeout time.Duration) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("WaitForColor(%d, %d)", x, y),
		execute: func() error {
			start := time.Now()
			for {
				ab.bot.CV().InvalidateCache()

				matches, err := ab.bot.CV().CheckColor(x, y, expected, tolerance)
				if err != nil {
					return fmt.Errorf("failed to check color: %w", err)
				}

				if matches {
					return nil
				}

				if time.Since(start) > timeout {
					return fmt.Errorf("color did not match within timeout")
				}

				time.Sleep(50 * time.Millisecond)
			}
		},
		timeout: timeout,
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// IfTemplateExistsRun conditionally executes a pre-built ActionBuilder if template is found
// This allows you to build the actions separately without a callback:
//
//	clickButton := l.Action().Click(100, 200).Sleep(1*time.Second)
//	l.Action().IfTemplateExistsRun(templates.Button, clickButton).Execute()
func (ab *ActionBuilder) IfTemplateExists(template cv.Template, then *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("IfTemplateExists(%s)", template.Name),
		execute: func() error {
			threshold := template.Threshold
			if threshold == 0 {
				threshold = 0.8
			}

			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			result, err := ab.bot.CV().FindTemplate(template.Name, config)
			if err != nil {
				return nil // Ignore errors, just skip
			}

			if result.Found {
				// Execute the pre-built actions directly
				return then.Execute()
			}

			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func (ab *ActionBuilder) IfTemplateExistsClick(template cv.Template) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("IfTemplateExists(%s)", template.Name),
		execute: func() error {
			threshold := template.Threshold
			if threshold == 0 {
				threshold = 0.8
			}

			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			result, err := ab.bot.CV().FindTemplate(template.Name, config)
			if err != nil {
				return nil // Ignore errors, just skip
			}

			if result.Found {
				// Execute the pre-built actions directly
				X, Y := regionCenter(*template.Region)
				return ab.bot.ADB().Click(X, Y)
			}

			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func (ab *ActionBuilder) IfTemplateExistsClickPoint(template cv.Template, X int, Y int) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("IfTemplateExists(%s)", template.Name),
		execute: func() error {
			threshold := template.Threshold
			if threshold == 0 {
				threshold = 0.8
			}

			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: threshold,
			}

			result, err := ab.bot.CV().FindTemplate(template.Name, config)
			if err != nil {
				return nil // Ignore errors, just skip
			}

			if result.Found {
				// Execute the pre-built actions directly
				return ab.bot.ADB().Click(X, Y)
			}

			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// UntilTemplateAppearsRun repeats a pre-built ActionBuilder's steps until template appears
// This allows you to pass an ActionBuilder directly without a callback:
//
//	clickAction := l.Action().Click(145, 234).Sleep(1*time.Second)
//	err := l.Action().UntilTemplateAppearsRun(templates.Shop, clickAction, 45).Execute()
//
// The ActionBuilder's steps will be re-executed on each loop iteration
func (ab *ActionBuilder) UntilTemplateAppears(template cv.Template, actions *ActionBuilder, maxAttempts int) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("UntilTemplateAppears(%s)", buildTemplatePath(template.Name)),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: template.Threshold,
			}

			for attempt := 0; attempt < maxAttempts; attempt++ {
				ab.bot.CV().InvalidateCache()

				result, err := ab.bot.CV().FindTemplate(buildTemplatePath(template.Name), config)
				if err == nil && result.Found {
					return nil // Template appeared!
				}

				// Re-execute the action builder's steps
				// We create a new builder with the same steps to re-execute them
				subBuilder := &ActionBuilder{
					bot:   ab.bot,
					ctx:   ab.ctx,
					steps: actions.steps, // Copy the steps
				}
				if err := subBuilder.Execute(); err != nil {
					return fmt.Errorf("action failed on attempt %d: %w", attempt+1, err)
				}

				time.Sleep(500 * time.Millisecond)
			}

			return fmt.Errorf("template did not appear after %d attempts", maxAttempts)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// UntilTemplateAppearsClick repeats clicking at the template's region center until the template appears
// This is a convenience method for the common pattern of clicking at a fixed point until something appears:
//
//	err := l.Action().UntilTemplateAppearsClick(templates.Main, 45).Execute()
//
// This is equivalent to:
//
//	X, Y := regionCenter(templates.Main.Region)
//	l.Action().UntilTemplateAppearsRun(templates.Main, l.Action().Click(X, Y).Sleep(1*time.Second), 45).Execute()
func (ab *ActionBuilder) UntilTemplateAppearsClick(template cv.Template, maxAttempts int) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("UntilTemplateAppearsClick(%s)", buildTemplatePath(template.Name)),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: template.Threshold,
			}

			X, Y := regionCenter(*template.Region)

			for attempt := 0; attempt < maxAttempts; attempt++ {
				ab.bot.CV().InvalidateCache()

				result, err := ab.bot.CV().FindTemplate(buildTemplatePath(template.Name), config)
				if err == nil && result.Found {
					return nil // Template appeared!
				}

				// Click and sleep
				if err := ab.bot.ADB().Click(X, Y); err != nil {
					return fmt.Errorf("click failed on attempt %d: %w", attempt+1, err)
				}

				time.Sleep(1 * time.Second)
			}

			return fmt.Errorf("template did not appear after %d attempts", maxAttempts)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// UntilTemplateAppearsClickPoint repeats clicking at a specific point until the template appears
// This is a convenience method for the common pattern of clicking at a fixed point until something appears:
//
//	err := l.Action().UntilTemplateAppearsClickPoint(templates.Main, 143, 360, 45).Execute()
//
// This is equivalent to:
//
//	l.Action().UntilTemplateAppearsRun(templates.Main, l.Action().Click(143, 360).Sleep(1*time.Second), 45).Execute()
func (ab *ActionBuilder) UntilTemplateAppearsClickPoint(template cv.Template, X int, Y int, maxAttempts int) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("UntilTemplateAppearsClickPoint(%s, %d, %d)", buildTemplatePath(template.Name), X, Y),
		execute: func() error {
			config := &cv.MatchConfig{
				Method:    cv.MatchMethodSSD,
				Threshold: template.Threshold,
			}

			for attempt := 0; attempt < maxAttempts; attempt++ {
				ab.bot.CV().InvalidateCache()

				result, err := ab.bot.CV().FindTemplate(buildTemplatePath(template.Name), config)
				if err == nil && result.Found {
					return nil // Template appeared!
				}

				// Click and sleep
				if err := ab.bot.ADB().Click(X, Y); err != nil {
					return fmt.Errorf("click failed on attempt %d: %w", attempt+1, err)
				}

				time.Sleep(1 * time.Second)
			}

			return fmt.Errorf("template did not appear after %d attempts", maxAttempts)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// InvalidateCache forces fresh capture on next CV operation
func (ab *ActionBuilder) InvalidateCache() *ActionBuilder {
	step := Step{
		name: "InvalidateCache",
		execute: func() error {
			ab.bot.CV().InvalidateCache()
			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func buildTemplatePath(name string) string {
	return fmt.Sprintf("templates/%s.png", name)
}

// Helper function to load template image (for getting dimensions)
func loadTemplateImage(name string) (image.Image, error) {
	path := buildTemplatePath(name)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode template: %w", err)
	}

	return img, nil
}

func regionCenter(region cv.Region) (int, int) {
	return region.X1 + (region.X2-region.X1)/2, region.Y1 + (region.Y2-region.Y1)/2
}

// buildMatchConfig creates a MatchConfig from a Template, applying region and threshold
func buildMatchConfig(template cv.Template, method cv.MatchMethod) *cv.MatchConfig {
	threshold := template.Threshold
	if threshold == 0 {
		threshold = 0.8
	}

	config := &cv.MatchConfig{
		Method:    method,
		Threshold: threshold,
	}

	// Apply template's search region if defined
	if template.Region != nil {
		config.SearchRegion = &image.Rectangle{
			Min: image.Point{X: template.Region.X1, Y: template.Region.Y1},
			Max: image.Point{X: template.Region.X2, Y: template.Region.Y2},
		}
	}

	return config
}
