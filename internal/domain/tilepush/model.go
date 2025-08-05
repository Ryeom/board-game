package tilepush

import "time"

type TileSet struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"unique;not null"`
	CreatedAt time.Time
	Tiles     []Tile
}

type Tile struct {
	ID        uint   `gorm:"primaryKey"`
	TileSetID uint   `gorm:"not null;index:idx_tile_set_id_shape,unique"`
	Shape     string `gorm:"not null;index:idx_tile_set_id_shape,unique"`
	ImageURL  string `gorm:"not null"`
	TileSet   TileSet
	CreatedAt time.Time
}
