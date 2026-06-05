package wsconnmgr

type ConnID string
type NodeID string

type Connection struct {
	ID   ConnID
	Node NodeID
	send chan []byte
}

type ConnectionManager interface {
	RegisterConn(node NodeID, conn Connection) error
	RemoveConn(connID ConnID) error

	GetConn(connID ConnID) (Connection, bool)
	ListConns(node NodeID) []ConnID

	Bind(node NodeID, connID ConnID) error
	Unbind(node NodeID, connID ConnID) error

	Send(connID ConnID, msg []byte) error
	Broadcast(node NodeID, msg []byte) error
}
