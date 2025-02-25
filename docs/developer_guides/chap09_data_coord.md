

## 8. Data Service



#### 8.1 Overview

<img src="./figs/data_coord.png" width=700>

#### 8.2 Data Service Interface

```go
type DataCoord interface {
	Component
	TimeTickProvider

	Flush(ctx context.Context, req *datapb.FlushRequest) (*datapb.FlushResponse, error)

	AssignSegmentID(ctx context.Context, req *datapb.AssignSegmentIDRequest) (*datapb.AssignSegmentIDResponse, error)
	GetSegmentStates(ctx context.Context, req *datapb.GetSegmentStatesRequest) (*datapb.GetSegmentStatesResponse, error)
	GetInsertBinlogPaths(ctx context.Context, req *datapb.GetInsertBinlogPathsRequest) (*datapb.GetInsertBinlogPathsResponse, error)
	GetSegmentInfoChannel(ctx context.Context) (*milvuspb.StringResponse, error)
	GetCollectionStatistics(ctx context.Context, req *datapb.GetCollectionStatisticsRequest) (*datapb.GetCollectionStatisticsResponse, error)
	GetPartitionStatistics(ctx context.Context, req *datapb.GetPartitionStatisticsRequest) (*datapb.GetPartitionStatisticsResponse, error)
	GetSegmentInfo(ctx context.Context, req *datapb.GetSegmentInfoRequest) (*datapb.GetSegmentInfoResponse, error)
	GetRecoveryInfo(ctx context.Context, req *datapb.GetRecoveryInfoRequest) (*datapb.GetRecoveryInfoResponse, error)
	SaveBinlogPaths(ctx context.Context, req *datapb.SaveBinlogPathsRequest) (*commonpb.Status, error)
	GetFlushedSegments(ctx context.Context, req *datapb.GetFlushedSegmentsRequest) (*datapb.GetFlushedSegmentsResponse, error)

	GetMetrics(ctx context.Context, req *milvuspb.GetMetricsRequest) (*milvuspb.GetMetricsResponse, error)
}
```



* *MsgBase*

```go
type MsgBase struct {
	MsgType   MsgType
	MsgID	    UniqueID
	Timestamp Timestamp
	SourceID  UniqueID
}
```

* *Flush*

```go
type FlushRequest struct {
	Base         *commonpb.MsgBase
	DbID         UniqueID
	CollectionID UniqueID
}
```

* *AssignSegmentID*

```go
type SegmentIDRequest struct {
	Count         uint32
	ChannelName   string
	CollectionID  UniqueID
	PartitionID   UniqueID
}

type AssignSegmentIDRequest struct {
	NodeID               int64               
	PeerRole             string              
	SegmentIDRequests    []*SegmentIDRequest 
}

type SegIDAssignment struct {
	SegID         UniqueID
	ChannelName   string
	Count         uint32
	CollectionID  UniqueID
	PartitionID   UniqueID
	ExpireTime    uint64
	Status        *commonpb.Status
}

type AssignSegmentIDResponse struct {
	SegIDAssignments []*SegmentIDAssignment
	Status           *commonpb.Status
}
```


* *GetSegmentStates*

```go
type GetSegmentStatesRequest struct {
	Base                 *commonpb.MsgBase 
	SegmentIDs           []int64           
}

type SegmentState int32

const (
	SegmentState_SegmentStateNone SegmentState = 0
	SegmentState_NotExist         SegmentState = 1
	SegmentState_Growing          SegmentState = 2
	SegmentState_Sealed           SegmentState = 3
	SegmentState_Flushed          SegmentState = 4
	SegmentState_Flushing         SegmentState = 5
)

type SegmentStateInfo struct {
	SegmentID     UniqueID
	State         commonpb.SegmentState
	StartPosition *internalpb.MsgPosition
	EndPosition   *internalpb.MsgPosition
	Status        *commonpb.Status
}

type GetSegmentStatesResponse struct {
	Status *commonpb.Status
	States []*SegmentStateInfo
}
```

* *GetInsertBinlogPaths*

```go
type GetInsertBinlogPathsRequest struct {
	Base      *commonpb.MsgBase
	SegmentID UniqueID
}

type GetInsertBinlogPathsResponse struct {
	FieldIDs []int64
	Paths    []*internalpb.StringList
	Status   *commonpb.Status
}
```

* *GetCollectionStatistics*

```go
type GetCollectionStatisticsRequest struct {
	Base         *commonpb.MsgBase
	DbID         int64
	CollectionID int64
}

type GetCollectionStatisticsResponse struct {
	Stats  []*commonpb.KeyValuePair
	Status *commonpb.Status
}
```

* *GetPartitionStatistics*

```go
type GetPartitionStatisticsRequest struct {
	Base         *commonpb.MsgBase
	DbID         UniqueID
	CollectionID UniqueID
	PartitionID  UniqueID
}

type GetPartitionStatisticsResponse struct {
	Stats  []*commonpb.KeyValuePair
	Status *commonpb.Status
}
```

* *GetSegmentInfo*

```go
type GetSegmentInfoRequest  struct{
	Base       *commonpb.MsgBase
	SegmentIDs []UniqueID
}

type SegmentInfo struct {
	ID                   int64                   
	CollectionID         int64                   
	PartitionID          int64                   
	InsertChannel        string                  
	NumOfRows            int64                   
	State                commonpb.SegmentState   
	DmlPosition          *internalpb.MsgPosition 
	MaxRowNum            int64                   
	LastExpireTime       uint64                  
	StartPosition        *internalpb.MsgPosition 
}

type GetSegmentInfoResponse  struct{
	Status *commonpb.Status
	infos  []SegmentInfo
}
```

* *GetRecoveryInfo*

```go
type GetRecoveryInfoRequest struct {
	Base                 *commonpb.MsgBase 
	CollectionID         int64             
	PartitionID          int64             
}


type VchannelInfo struct {
	CollectionID         int64                   
	ChannelName          string                  
	SeekPosition         *internalpb.MsgPosition 
	UnflushedSegments    []*SegmentInfo          
	FlushedSegments      []int64                 
}

type SegmentBinlogs struct {
	SegmentID            int64          
	FieldBinlogs         []*FieldBinlog 
}

type GetRecoveryInfoResponse struct {
	Status               *commonpb.Status  
	Channels             []*VchannelInfo   
	Binlogs              []*SegmentBinlogs 
}
```

* *SaveBinlogPaths*
```go
type SegmentStartPosition struct {
	StartPosition        *internalpb.MsgPosition 
	SegmentID            int64                   
}

type SaveBinlogPathsRequest struct {
	Base                 *commonpb.MsgBase       
	SegmentID            int64                   
	CollectionID         int64                   
	Field2BinlogPaths    []*ID2PathList          
	CheckPoints          []*CheckPoint           
	StartPositions       []*SegmentStartPosition 
	Flushed              bool                    
}
```




#### 8.2 Insert Channel

* *InsertMsg*

```go
type InsertRequest struct {
	Base           *commonpb.MsgBase
	DbName         string
	CollectionName string
	PartitionName  string
	DbID           UniqueID
	CollectionID   UniqueID
	PartitionID    UniqueID
	SegmentID      UniqueID
	ChannelID      string
	Timestamps     []uint64
	RowIDs         []int64
	RowData        []*commonpb.Blob
}

type InsertMsg struct {
	BaseMsg
	InsertRequest
}
```



#### 8.2 Data Node Interface

```go
type DataNode interface {
	Component

	WatchDmChannels(ctx context.Context, req *datapb.WatchDmChannelsRequest) (*commonpb.Status, error)
	FlushSegments(ctx context.Context, req *datapb.FlushSegmentsRequest) (*commonpb.Status, error)
}
```

* *WatchDmChannels*

```go
type WatchDmChannelRequest struct {
	Base         *commonpb.MsgBase
	Vchannels    []*VchannelInfo
}
```

* *FlushSegments*

```go
type FlushSegmentsRequest struct {
	Base         *commonpb.MsgBase
	DbID         UniqueID
	CollectionID UniqueID
	SegmentIDs   []int64
}
```


#### 8.2 SegmentStatistics Update Channel

* *SegmentStatisticsMsg*

```go
type SegmentStatisticsUpdates struct {
	SegmentID     UniqueID
	MemorySize    int64
	NumRows       int64
	CreateTime    uint64
	EndTime       uint64
	StartPosition *internalpb.MsgPosition
	EndPosition   *internalpb.MsgPosition
}

type SegmentStatistics struct {
	Base                 *commonpb.MsgBase
	SegStats             []*SegmentStatisticsUpdates
}

type SegmentStatisticsMsg struct {
	BaseMsg
	SegmentStatistics
}

```
#### 8.3 DataNode Time Tick Channel

* *DataNode Tt Msg*

```go
message DataNodeTtMsg {
    Base        *commonpb.MsgBase
    ChannelName string
    Timestamp   uint64
}
```

