package core

type Handler = func(*Request, ResponseExtender, func())
