package persistence

type extensionInstallRow struct {
	ID           string `gorm:"primaryKey;column:id"`
	Manifest     string `gorm:"column:manifest;not null"`
	InstallMode  string `gorm:"column:install_mode;not null;default:linked;index:extension_installs_install_mode_idx"`
	RootPath     string `gorm:"column:root_path;not null;uniqueIndex"`
	ManifestPath string `gorm:"column:manifest_path;not null"`
	Integrity    string `gorm:"column:integrity;not null;index:extension_installs_integrity_idx"`
	Enabled      int    `gorm:"column:enabled;not null;default:0;index:extension_installs_enabled_idx"`
	Status       string `gorm:"column:status;not null"`
	Error        string `gorm:"column:error"`
	TimeCreated  string `gorm:"column:time_created;not null"`
	TimeUpdated  string `gorm:"column:time_updated;not null"`
}

func (extensionInstallRow) TableName() string { return "extension_installs" }
