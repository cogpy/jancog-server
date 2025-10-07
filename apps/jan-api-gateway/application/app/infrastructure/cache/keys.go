package cache

import "fmt"

func OrganizationModelsCacheKey(organizationID uint) string {
	return fmt.Sprintf(OrganizationModelsCacheKeyPattern, organizationID)
}

func ProjectModelsCacheKey(projectID uint) string {
	return fmt.Sprintf(ProjectModelsCacheKeyPattern, projectID)
}
