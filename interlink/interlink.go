package interlink

func Init() (err error) {
	srv := server{}
	err = srv.Init()
	if err != nil {
		return
	}

	return
}
