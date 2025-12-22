package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/icegreg/chat-smpl/pkg/postgres"
	"github.com/icegreg/chat-smpl/services/org/internal/model"
	"github.com/icegreg/chat-smpl/services/org/internal/repository"
	"github.com/icegreg/chat-smpl/services/org/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	databaseURL := getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5435/chatapp?sslmode=disable")

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(databaseURL))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	repo := repository.NewOrgRepository(pool)
	svc := service.NewOrgService(repo)

	switch os.Args[1] {
	case "generate":
		generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
		holdingName := generateCmd.String("name", "ООО Холдинг", "Name of the holding company")
		subsidiaryCount := generateCmd.Int("subsidiaries", 3, "Number of subsidiary companies")
		generateCmd.Parse(os.Args[2:])

		if err := generateHolding(ctx, svc, *holdingName, *subsidiaryCount); err != nil {
			log.Fatalf("Failed to generate holding: %v", err)
		}
		fmt.Println("Holding structure generated successfully!")

	case "link-users":
		if err := linkUsers(ctx, pool, svc); err != nil {
			log.Fatalf("Failed to link users: %v", err)
		}
		fmt.Println("Users linked to organization successfully!")

	case "list":
		listCmd := flag.NewFlagSet("list", flag.ExitOnError)
		listType := listCmd.String("type", "companies", "Type to list: companies, departments, positions, employees")
		listCmd.Parse(os.Args[2:])

		if err := listEntities(ctx, svc, *listType); err != nil {
			log.Fatalf("Failed to list: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`org-cli - Organization Management CLI

Usage:
  org-cli <command> [options]

Commands:
  generate     Generate holding company structure
    -name      Holding company name (default: "ООО Холдинг")
    -subsidiaries  Number of subsidiary companies (default: 3)

  link-users   Link existing users (except guests) to random departments/positions

  list         List entities
    -type      Type to list: companies, departments, positions, employees

Environment:
  DATABASE_URL  PostgreSQL connection string (default: postgres://chatapp:secret@localhost:5435/chatapp?sslmode=disable)
`)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// generateHolding creates a complete holding company structure
func generateHolding(ctx context.Context, svc service.OrgService, holdingName string, subsidiaryCount int) error {
	fmt.Printf("Generating holding structure: %s with %d subsidiaries...\n", holdingName, subsidiaryCount)

	// Create holding company
	holding := &model.Company{
		ID:       uuid.New(),
		Name:     holdingName,
		Timezone: "Europe/Moscow",
	}
	shortName := "Холдинг"
	holding.ShortName = &shortName

	if err := svc.CreateCompany(ctx, holding); err != nil {
		return fmt.Errorf("failed to create holding: %w", err)
	}
	fmt.Printf("  Created holding: %s (%s)\n", holding.Name, holding.ID)

	// Subsidiary names
	subsidiaryNames := []string{
		"ООО Производство Альфа",
		"ООО Логистика Бета",
		"ООО ИТ Гамма",
		"ООО Финансы Дельта",
		"ООО Торговля Эпсилон",
	}

	// Create positions for holding
	positions := createPositions(ctx, svc, holding.ID)
	if len(positions) == 0 {
		return fmt.Errorf("failed to create positions")
	}

	// Create departments for holding
	createDepartments(ctx, svc, holding.ID, positions)

	// Create subsidiaries
	for i := 0; i < subsidiaryCount && i < len(subsidiaryNames); i++ {
		subsidiary := &model.Company{
			ID:       uuid.New(),
			ParentID: &holding.ID,
			Name:     subsidiaryNames[i],
			Timezone: "Europe/Moscow",
		}
		if err := svc.CreateCompany(ctx, subsidiary); err != nil {
			fmt.Printf("  Warning: failed to create subsidiary %s: %v\n", subsidiaryNames[i], err)
			continue
		}
		fmt.Printf("  Created subsidiary: %s (%s)\n", subsidiary.Name, subsidiary.ID)

		// Create positions for subsidiary
		subPositions := createPositions(ctx, svc, subsidiary.ID)

		// Create departments for subsidiary
		createDepartments(ctx, svc, subsidiary.ID, subPositions)
	}

	return nil
}

func createPositions(ctx context.Context, svc service.OrgService, companyID uuid.UUID) []model.Position {
	positionDefs := []struct {
		Name      string
		ShortName string
		Level     int
	}{
		{"Генеральный директор", "ГД", 100},
		{"Директор", "Дир", 90},
		{"Заместитель директора", "ЗД", 80},
		{"Начальник отдела", "НО", 70},
		{"Заместитель начальника отдела", "ЗНО", 60},
		{"Ведущий специалист", "ВС", 50},
		{"Старший специалист", "СС", 40},
		{"Специалист", "Спец", 30},
		{"Младший специалист", "МС", 20},
		{"Стажёр", "Стаж", 10},
	}

	var positions []model.Position
	for _, pd := range positionDefs {
		pos := &model.Position{
			ID:        uuid.New(),
			CompanyID: companyID,
			Name:      pd.Name,
			Level:     pd.Level,
		}
		shortName := pd.ShortName
		pos.ShortName = &shortName

		if err := svc.CreatePosition(ctx, pos); err != nil {
			fmt.Printf("    Warning: failed to create position %s: %v\n", pd.Name, err)
			continue
		}
		positions = append(positions, *pos)
	}
	fmt.Printf("    Created %d positions\n", len(positions))
	return positions
}

func createDepartments(ctx context.Context, svc service.OrgService, companyID uuid.UUID, positions []model.Position) {
	departmentDefs := []struct {
		Name      string
		ShortName string
		Children  []struct {
			Name      string
			ShortName string
		}
	}{
		{
			Name:      "Администрация",
			ShortName: "Адм",
			Children: []struct {
				Name      string
				ShortName string
			}{
				{"Секретариат", "Секр"},
				{"Юридический отдел", "Юр"},
			},
		},
		{
			Name:      "Финансовый департамент",
			ShortName: "Фин",
			Children: []struct {
				Name      string
				ShortName string
			}{
				{"Бухгалтерия", "Бух"},
				{"Планово-экономический отдел", "ПЭО"},
				{"Казначейство", "Казн"},
			},
		},
		{
			Name:      "Департамент информационных технологий",
			ShortName: "ДИТ",
			Children: []struct {
				Name      string
				ShortName string
			}{
				{"Отдел разработки", "Разр"},
				{"Отдел инфраструктуры", "Инфр"},
				{"Отдел технической поддержки", "ТП"},
			},
		},
		{
			Name:      "Департамент кадров",
			ShortName: "HR",
			Children: []struct {
				Name      string
				ShortName string
			}{
				{"Отдел подбора персонала", "Рекр"},
				{"Отдел развития персонала", "Разв"},
			},
		},
		{
			Name:      "Производственный департамент",
			ShortName: "Произв",
			Children: []struct {
				Name      string
				ShortName string
			}{
				{"Цех № 1", "Цех1"},
				{"Цех № 2", "Цех2"},
				{"Отдел контроля качества", "ОКК"},
			},
		},
		{
			Name:      "Коммерческий департамент",
			ShortName: "Комм",
			Children: []struct {
				Name      string
				ShortName string
			}{
				{"Отдел продаж", "Прод"},
				{"Отдел маркетинга", "Марк"},
				{"Отдел закупок", "Зак"},
			},
		},
	}

	sortOrder := 0
	for _, dd := range departmentDefs {
		dept := &model.Department{
			ID:        uuid.New(),
			CompanyID: companyID,
			Name:      dd.Name,
			SortOrder: sortOrder,
		}
		shortName := dd.ShortName
		dept.ShortName = &shortName
		sortOrder++

		if err := svc.CreateDepartment(ctx, dept); err != nil {
			fmt.Printf("    Warning: failed to create department %s: %v\n", dd.Name, err)
			continue
		}

		// Create child departments
		for _, child := range dd.Children {
			childDept := &model.Department{
				ID:                 uuid.New(),
				CompanyID:          companyID,
				ParentDepartmentID: &dept.ID,
				Name:               child.Name,
				SortOrder:          sortOrder,
			}
			childShortName := child.ShortName
			childDept.ShortName = &childShortName
			sortOrder++

			if err := svc.CreateDepartment(ctx, childDept); err != nil {
				fmt.Printf("    Warning: failed to create child department %s: %v\n", child.Name, err)
			}
		}
	}
	fmt.Printf("    Created departments hierarchy\n")
}

// linkUsers links existing users to random departments and positions
func linkUsers(ctx context.Context, pool *pgxpool.Pool, svc service.OrgService) error {
	fmt.Println("Linking users to organization structure...")

	// Get all non-guest users
	rows, err := pool.Query(ctx, `
		SELECT id, username FROM con_test.users
		WHERE role != 'guest'
	`)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	type userInfo struct {
		ID       uuid.UUID
		Username string
	}
	var users []userInfo
	for rows.Next() {
		var u userInfo
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if len(users) == 0 {
		fmt.Println("No users found to link")
		return nil
	}
	fmt.Printf("Found %d users to link\n", len(users))

	// Get all companies
	companies, _, err := svc.ListCompanies(ctx, nil, false, 1, 100)
	if err != nil {
		return fmt.Errorf("failed to list companies: %w", err)
	}
	if len(companies) == 0 {
		return fmt.Errorf("no companies found. Run 'generate' first")
	}

	// Collect all departments and positions
	var allDepartments []model.Department
	var allPositions []model.Position

	for _, company := range companies {
		depts, _, err := svc.ListDepartments(ctx, company.ID, nil, false, 1, 100)
		if err != nil {
			continue
		}
		allDepartments = append(allDepartments, depts...)

		positions, _, err := svc.ListPositions(ctx, company.ID, false, 1, 100)
		if err != nil {
			continue
		}
		allPositions = append(allPositions, positions...)
	}

	if len(allDepartments) == 0 || len(allPositions) == 0 {
		return fmt.Errorf("no departments or positions found. Run 'generate' first")
	}

	fmt.Printf("Found %d departments and %d positions\n", len(allDepartments), len(allPositions))

	// Link each user
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	linkedCount := 0

	for _, user := range users {
		// Check if already linked
		existing, err := svc.GetEmployeeByUserID(ctx, user.ID, true)
		if err == nil && existing != nil {
			fmt.Printf("  User %s already linked, skipping\n", user.Username)
			continue
		}

		// Pick random department and matching position
		dept := allDepartments[rng.Intn(len(allDepartments))]

		// Find positions for this company
		var companyPositions []model.Position
		for _, pos := range allPositions {
			if pos.CompanyID == dept.CompanyID {
				companyPositions = append(companyPositions, pos)
			}
		}
		if len(companyPositions) == 0 {
			continue
		}
		pos := companyPositions[rng.Intn(len(companyPositions))]

		emp := &model.Employee{
			ID:           uuid.New(),
			UserID:       user.ID,
			DepartmentID: dept.ID,
			PositionID:   pos.ID,
			IsPrimary:    true,
		}

		if err := svc.CreateEmployee(ctx, emp); err != nil {
			fmt.Printf("  Warning: failed to link user %s: %v\n", user.Username, err)
			continue
		}

		linkedCount++
		fmt.Printf("  Linked %s to %s as %s\n", user.Username, dept.Name, pos.Name)
	}

	fmt.Printf("Linked %d users\n", linkedCount)
	return nil
}

// listEntities lists entities of specified type
func listEntities(ctx context.Context, svc service.OrgService, entityType string) error {
	switch entityType {
	case "companies":
		companies, total, err := svc.ListCompanies(ctx, nil, true, 1, 100)
		if err != nil {
			return err
		}
		fmt.Printf("Companies (%d):\n", total)
		for _, c := range companies {
			parent := ""
			if c.ParentID != nil {
				parent = fmt.Sprintf(" (parent: %s)", *c.ParentID)
			}
			active := ""
			if !c.IsActive {
				active = " [INACTIVE]"
			}
			fmt.Printf("  %s - %s%s%s\n", c.ID, c.Name, parent, active)
		}

	case "departments":
		companies, _, err := svc.ListCompanies(ctx, nil, false, 1, 100)
		if err != nil {
			return err
		}
		for _, company := range companies {
			depts, total, err := svc.ListDepartments(ctx, company.ID, nil, true, 1, 100)
			if err != nil {
				continue
			}
			fmt.Printf("\nDepartments for %s (%d):\n", company.Name, total)
			for _, d := range depts {
				parent := ""
				if d.ParentDepartmentID != nil {
					parent = fmt.Sprintf(" (parent: %s)", *d.ParentDepartmentID)
				}
				fmt.Printf("  %s - %s%s\n", d.ID, d.Name, parent)
			}
		}

	case "positions":
		companies, _, err := svc.ListCompanies(ctx, nil, false, 1, 100)
		if err != nil {
			return err
		}
		for _, company := range companies {
			positions, total, err := svc.ListPositions(ctx, company.ID, true, 1, 100)
			if err != nil {
				continue
			}
			fmt.Printf("\nPositions for %s (%d):\n", company.Name, total)
			for _, p := range positions {
				fmt.Printf("  %s - %s (level: %d)\n", p.ID, p.Name, p.Level)
			}
		}

	case "employees":
		employees, total, err := svc.ListEmployees(ctx, nil, nil, true, 1, 100)
		if err != nil {
			return err
		}
		fmt.Printf("Employees (%d):\n", total)
		for _, e := range employees {
			fmt.Printf("  %s - user: %s, dept: %s, pos: %s\n", e.ID, e.UserID, e.DepartmentID, e.PositionID)
		}

	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	return nil
}
