package archiverMedia

import (
	"sync"

	"github.com/fanap-infra/archiverMedia/internal/virtualMedia"

	"github.com/fanap-infra/fsEngine"
	"github.com/fanap-infra/log"
)

type Archiver struct {
	log           *log.Logger
	EventsHandler Events
	fs            *fsEngine.FSEngine
	blockSize     uint32
	crudMutex     sync.Mutex
	openFiles     map[uint32]*virtualMedia.VirtualMedia
}
