package main

type Connection struct {
	Name       string
	OriginID   string
	OriginType string
	DestID     string
	DestType   string
	OriginName string
	DestName   string
}

var connections = []Connection{
	{
		Name:       "Heim -> Büro",
		OriginID:   "de:07111:010240",
		OriginType: "stop",
		DestID:     "de:07111:010001",
		DestType:   "stop",
		OriginName: "Franz-Weis-Straße, Koblenz - Rauental",
		DestName:   "Koblenz Hauptbahnhof, Koblenz",
	},
	{
		Name:       "Büro -> Heim",
		OriginID:   "de:07111:010001",
		OriginType: "stop",
		DestID:     "de:07111:010240",
		DestType:   "stop",
		OriginName: "Koblenz Hauptbahnhof, Koblenz",
		DestName:   "Franz-Weis-Straße, Koblenz - Rauental",
	},
}
