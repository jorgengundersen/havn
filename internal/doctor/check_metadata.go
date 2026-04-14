package doctor

import "time"

type checkMetadata struct {
	id            string
	tier          string
	container     string
	prerequisites []string
	timeout       time.Duration
}

func newHostCheckMetadata(id string, prerequisites []string, timeout time.Duration) checkMetadata {
	return newCheckMetadata(id, "host", "", prerequisites, timeout)
}

func newContainerCheckMetadata(id, container string, prerequisites []string, timeout time.Duration) checkMetadata {
	return newCheckMetadata(id, "container", container, prerequisites, timeout)
}

func newCheckMetadata(id, tier, container string, prerequisites []string, timeout time.Duration) checkMetadata {
	clonedPrerequisites := make([]string, len(prerequisites))
	copy(clonedPrerequisites, prerequisites)

	return checkMetadata{
		id:            id,
		tier:          tier,
		container:     container,
		prerequisites: clonedPrerequisites,
		timeout:       timeout,
	}
}

func (m checkMetadata) ID() string {
	return m.id
}

func (m checkMetadata) Tier() string {
	return m.tier
}

func (m checkMetadata) Container() string {
	return m.container
}

func (m checkMetadata) Prerequisites() []string {
	if len(m.prerequisites) == 0 {
		return nil
	}

	clonedPrerequisites := make([]string, len(m.prerequisites))
	copy(clonedPrerequisites, m.prerequisites)

	return clonedPrerequisites
}

func (m checkMetadata) Timeout() time.Duration {
	return m.timeout
}
