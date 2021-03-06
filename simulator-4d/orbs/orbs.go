/*
	天体及天体计算框架
*/
package orbs

import (
	"fmt"
	"log"
	"math"
	"math/rand"
)

// 天体结构体声明
type Orb struct {
	X    float64 `json:"x"`  // 坐标x
	Y    float64 `json:"y"`  // 坐标y
	Z    float64 `json:"z"`  // 坐标z
	W    float64 `json:"w"`  // 坐标w
	Vx   float64 `json:"vx"` // 速度x
	Vy   float64 `json:"vy"` // 速度y
	Vz   float64 `json:"vz"` // 速度z
	Vw   float64 `json:"vw"` // 速度w
	Mass float64 `json:"m"`  // 质量
	Id   int32   `json:"i"`  // Id<0表示状态不正常 不能参与计算,不能当下标使用,只能参与比较
	//Stat int32   `json:"st"` // 用于标记是否已爆炸 1=正常 2=已爆炸 //作废 Id instead
	//Size int     `json:"sz"` // 大小，用于计算吞并的天体数量 //作废 Mass instead
	//idx       int
	//crashedBy int
}

// 加速度结构体
type Acc struct {
	Ax float64
	Ay float64
	Az float64
	Aw float64
	A  float64
}

// 配置
type InitConfig struct {
	Mass         float64
	Wide         float64
	Velo         float64
	Arrange      int     // 分布方式 0=线性 1=立方体 2=圆盘圆柱 3=球形
	Assemble     int     // 聚集方式：0=均匀分布 1=中心靠拢开方分布 2=比例加权分布 3=比例立方
	BigMass      float64 // 大块头的质量 比如处于中心的黑洞
	BigNum       int     // 大块头个数
	BigDistStyle int     // big mass orb distribute style: 0=center 1=outer edge 2=middle of one radius 3=random

}

// 碰撞事件
type CrashEvent struct {
	Idx       int
	CrashedBy int
}

// 万有引力常数
const G = 0.000005

// 最小天体距离值 两天体距离小于此值了会相撞
const MIN_CRITICAL_DIST = 2

// 监控速度和加速度
var maxVeloX, maxVeloY, maxVeloZ, maxVeloW, maxAccX, maxAccY, maxAccZ, maxAccW, maxMass, allMass, allWC float64 = 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
var maxMassId int32 = 0
var clearTimes, willTimes, realTimes int64 = 0, 0, 0

var c chan int                     //= make(chan int, 10000)	// orb.update()完成队列
var crashEventChan chan CrashEvent //= make(chan CrashEvent, 0) // 撞击事件队列

var nCount, nCrashed int = 0, 0

// 初始化天体位置，质量，加速度 在一片区域随机分布
func InitOrbs(num int, config *InitConfig) []Orb {
	oList := make([]Orb, num)

	// 通用属性设置
	for i := 0; i < num; i++ {
		o := &oList[i]
		o.Mass = rand.Float64() * config.Mass
		o.Id = int32(i + 1) // rand.Int()
		//o.Stat = 1
		allMass += o.Mass
	}

	// 排列分布 与 中心聚集
	switch config.Arrange {
	case 0: //线性
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}

			o := &oList[i]

			// 沿x轴分布
			o.X = (0.5 - rand.Float64()) * wide
			o.Y, o.Z, o.W = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0

			if o.X < 0 {
				o.Vx = (1.0 + rand.Float64()) * config.Velo
				o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			} else {
				o.Vx = -(1.0 + rand.Float64()) * config.Velo
				o.Vy = (1.0 + rand.Float64()) * config.Velo
			}
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
			o.Vw = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
		}
	case 1: //立方体
		for i := 0; i < num; i++ {
			o := &oList[i]
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o.X = (0.5 - rand.Float64()) * wide
			o.Y = (0.5 - rand.Float64()) * wide
			o.Z = (0.5 - rand.Float64()) * wide
			o.W = (0.5 - rand.Float64()) * wide

			o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vw = (rand.Float64() - 0.5) * config.Velo * 2.0
		}
	case 2: //圆盘 随机选经度 随机选半径 随机选高低 刻意降低垂直于柱面的速度
		for i := 0; i < num; i++ {
			o := &oList[i]
			long := rand.Float64() * math.Pi * 2
			high := (0.5 - rand.Float64()) * config.Wide
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 3.0)
			case 4:
				wide = config.Wide * math.Pow(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 4.0)
			default:
				wide = config.Wide
			}
			// 分布在x,y平面的盘
			radius := wide / 2.0 * math.Sqrt(rand.Float64())
			o.X, o.Y = math.Cos(long)*radius, math.Sin(long)*radius
			o.Z = high / 256.0
			o.W = high / 256.0

			//o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0 * math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			//o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0 * math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vx = math.Cos(long+math.Pi/2.0) * config.Velo * 2.0 //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vy = math.Sin(long+math.Pi/2.0) * config.Velo * 2.0 //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
			o.Vw = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
		}
	case 3: //球形
		//方法一： 随机经度 随机半径 随机高度*sin(半径) 产生的数据从y轴上方看z面，不均匀
		//方法二： 随机经度 随机纬度=acos(rand(0-1))
		for i := 0; i < num; i++ {
			o := &oList[i]
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100))
			case 2:
				wide = config.Wide * math.Pow(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 3.0)
			case 4:
				wide = config.Wide * math.Pow(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 4.0)
			default:
				wide = config.Wide
			}
			long := rand.Float64() * math.Pi * 2
			lati := math.Acos(rand.Float64()*2.0 - 1.0)
			radius := math.Pow(rand.Float64(), 1.0/3.0) * wide / 2.0
			o.X, o.Y = radius*math.Cos(long)*math.Sin(lati), radius*math.Sin(long)*math.Sin(lati)
			o.Z = radius * math.Cos(lati)
			// 四维球体不知道怎么均衡分布
			o.W = radius * math.Cos(lati)
			o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vw = (rand.Float64() - 0.5) * config.Velo * 2.0
		}
	case 4: //线性 4轴
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]

			o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0
			o.Vx, o.Vy, o.Vz = (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0
			award := i % 2

			switch award {
			case 0:
				o.X = (0.5 - rand.Float64()) * wide
				if o.X < 0 {
					o.Vx = (1.0 + rand.Float64()) * config.Velo
					o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vx = -(1.0 + rand.Float64()) * config.Velo
					o.Vy = (1.0 + rand.Float64()) * config.Velo
				}
			case 1:
				o.Y = (0.5 - rand.Float64()) * wide
				if o.Y < 0 {
					o.Vy = (1.0 + rand.Float64()) * config.Velo
					o.Vz = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vy = -(1.0 + rand.Float64()) * config.Velo
					o.Vz = (1.0 + rand.Float64()) * config.Velo
				}
			case 2:
				o.Z = (0.5 - rand.Float64()) * wide
				if o.Z < 0 {
					o.Vz = (1.0 + rand.Float64()) * config.Velo
					o.Vx = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vz = -(1.0 + rand.Float64()) * config.Velo
					o.Vx = (1.0 + rand.Float64()) * config.Velo
				}
			default:
			}
		}
	case 5: //线性 6轴
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]

			o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0
			o.Vx, o.Vy, o.Vz = (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0
			award := i % 3

			switch award {
			case 0:
				o.X = (0.5 - rand.Float64()) * wide
				if o.X < 0 {
					o.Vx = (1.0 + rand.Float64()) * config.Velo
					o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vx = -(1.0 + rand.Float64()) * config.Velo
					o.Vy = (1.0 + rand.Float64()) * config.Velo
				}
			case 1:
				o.Y = (0.5 - rand.Float64()) * wide
				if o.Y < 0 {
					o.Vy = (1.0 + rand.Float64()) * config.Velo
					o.Vz = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vy = -(1.0 + rand.Float64()) * config.Velo
					o.Vz = (1.0 + rand.Float64()) * config.Velo
				}
			case 2:
				o.Z = (0.5 - rand.Float64()) * wide
				if o.Z < 0 {
					o.Vz = (1.0 + rand.Float64()) * config.Velo
					o.Vx = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vz = -(1.0 + rand.Float64()) * config.Velo
					o.Vx = (1.0 + rand.Float64()) * config.Velo
				}
			default:
			}
		}
	case 6: //线性 1轴
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]
			o.X = (rand.Float64()) * wide
			o.Y, o.Z, o.W = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0

			if o.X < 0 {
				o.Vx = (1.0 + rand.Float64()) * config.Velo
				o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			} else {
				o.Vx = -(1.0 + rand.Float64()) * config.Velo
				o.Vy = (1.0 + rand.Float64()) * config.Velo
			}
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
			o.Vw = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
		}
	default:
	}

	// 如果配置了大块头质量 0=中心 1=边缘 2=半径的中点 3=随机
	if config.BigMass != 0.0 {
		for i := 0; i < config.BigNum && config.BigNum <= len(oList); i++ {

			eternalOrb := &oList[num-1-i]
			allMass += config.BigMass - eternalOrb.Mass
			eternalOrb.Mass = config.BigMass
			eternalOrb.X, eternalOrb.Y, eternalOrb.Z = 0, 0, 0
			eternalOrb.Vx, eternalOrb.Vy, eternalOrb.Vz = 0, 0, 0
			switch config.BigDistStyle {
			case 1:
				// 环形分布
				eternalOrb.X = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0
				eternalOrb.Y = math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0
				// 逆时针运动
				eternalOrb.Vx = -math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
				eternalOrb.Vy = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
			case 2:
				eternalOrb.X = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 / 2.0
				eternalOrb.Y = math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 / 2.0
				eternalOrb.Vx = -math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
				eternalOrb.Vy = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
			case 3:
				eternalOrb.X = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 * rand.Float64()
				eternalOrb.Y = math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 * rand.Float64()
				eternalOrb.Vx = -math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
				eternalOrb.Vy = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
			case 0:
				fallthrough
			default:
			}
		}

	}
	return oList
}

func UpdateOrbs(oList []Orb, numTimes int) int64 {
	realTimes = 0
	//theListLength = len(oList)
	willTimes = int64(len(oList)) * int64(len(oList)) * int64(numTimes)
	// 初始化chan CrashEvent ,orb.update()将会往crashEventChan中push事件
	// 事件队列，提升效率 15%左右
	crashEventChan = make(chan CrashEvent, len(oList))
	// 分配足够的队列空间，提升效率 0.5%左右
	c = make(chan int, len(oList))

	for i := 0; i < numTimes; i++ {
		realTimes += UpdateOrbsOnce(oList, i)
	}
	return realTimes
}

// 所有天体运动一次
func UpdateOrbsOnce(oList []Orb, nStep int) int64 {
	thelen := len(oList)
	nCount := 0
	var o, target *Orb
	var targetMassOld float64
	for i := 0; i < thelen; i++ {
		//oList[i].idx = i
		//oList[i].crashedBy = -1
		go oList[i].Update(oList, i)
	}
	for {
		if nCount >= thelen {
			break
		}

		select {
		case <-c:
			// 正常计算完成任务返回
			nCount++
		case anEvent := <-crashEventChan:
			nCrashed++
			// 收集事件队列信息
			o = &oList[anEvent.Idx]
			// 只处理自己被谁撞击合并
			target = &oList[anEvent.CrashedBy]
			log.Println("a CrashEvent:", o.Id, "crashed by", target.Id, "index:", anEvent, "nCrashed:", nCrashed, "nStep:", nStep)
			targetMassOld = target.Mass
			target.Mass += o.Mass
			target.Vx = (targetMassOld*target.Vx + o.Mass*o.Vx) / target.Mass
			target.Vy = (targetMassOld*target.Vy + o.Mass*o.Vy) / target.Mass
			target.Vz = (targetMassOld*target.Vz + o.Mass*o.Vz) / target.Mass
			target.Vw = (targetMassOld*target.Vw + o.Mass*o.Vw) / target.Mass
			o.Mass = 0
			//o.Stat = 2
			//default:
			//	log.Println("nothing when select")
		}
	}
	return int64(thelen) * int64(nCount)
}

// 天体运动一次
func (o *Orb) Update(oList []Orb, idx int) {
	// 先把位置移动起来，再计算环境中的加速度，再更新速度，为了更好地解决并行计算数据同步问题
	if o.Id >= 0 /*o.Stat == 1*/ {
		aAll := o.CalcGravityAll(oList, idx)
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
		o.W += o.Vw
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
		o.Vw += aAll.Aw
		// 监控速度和加速度
		if maxVeloX < math.Abs(o.Vx) {
			maxVeloX = o.Vx
		}
		if maxVeloY < math.Abs(o.Vy) {
			maxVeloY = o.Vy
		}
		if maxVeloZ < math.Abs(o.Vz) {
			maxVeloZ = o.Vz
		}
		if maxVeloW < math.Abs(o.Vw) {
			maxVeloW = o.Vw
		}
		if maxAccX < math.Abs(aAll.Ax) {
			maxAccX = aAll.Ax
		}
		if maxAccY < math.Abs(aAll.Ay) {
			maxAccY = aAll.Ay
		}
		if maxAccZ < math.Abs(aAll.Az) {
			maxAccZ = aAll.Az
		}
		if maxAccW < math.Abs(aAll.Aw) {
			maxAccW = aAll.Aw
		}
		if maxMass < o.Mass {
			maxMass = o.Mass
			maxMassId = o.Id
		}
	}
	c <- idx //len(oList)
}

// 计算天体受到的总体引力
func (o *Orb) CalcGravityAll(oList []Orb, idx int) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		target := &oList[i]
		if /*target.Stat != 1 || o.Stat != 1*/ target.Id < 0 || o.Id < 0 || target.Id == o.Id {
			continue
		}

		dist := o.CalcDist(target)

		// 距离太近，被撞
		isTooNearly := dist*dist < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST
		// 速度太快，被撕裂 me ripped by ta
		isMeRipped := dist < math.Sqrt(o.Vx*o.Vx+o.Vy*o.Vy+o.Vz*o.Vz+o.Vw*o.Vw)*8

		if isTooNearly || isMeRipped {
			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass < target.Mass {
				// 碰撞事件交给主goroutine处理对方的质量改变，这里发送信息，不做修改操作
				o.Id = -o.Id //o.Stat = 2 // 此处必须对自己标记，否则会出现被多个ta撞击的事件
				crashEventChan <- CrashEvent{idx, i}
				//o.crashedBy = i // 不能取target.idx // 待思考为什么 协程间数据共享，不安全
				// 由于并发数据分离，当前goroutine只允许操作当前orb,不允许操作别的orb，所以不允许操作ta的数据
			}
			// no else 在循环时可能有多个o crashed ta,但是只有一个o crashed by ta
		} else {
			// 作用正常，累计计算受到的所有的万有引力
			gTmp := o.CalcGravity(&oList[i], dist)
			gAll.Ax += gTmp.Ax
			gAll.Ay += gTmp.Ay
			gAll.Az += gTmp.Az
			gAll.Aw += gTmp.Aw
		}
	}

	return gAll
}

// 计算天体与目标的引力
func (o *Orb) CalcGravity(target *Orb, dist float64) Acc {
	var a Acc
	// 万有引力公式
	a.A = target.Mass / (dist * dist) * G
	a.Ax = -a.A * (o.X - target.X) / dist //a.A * math.Cos(a.Dir)
	a.Ay = -a.A * (o.Y - target.Y) / dist //a.A * math.Sin(a.Dir)
	a.Az = -a.A * (o.Z - target.Z) / dist //a.A * math.Sin(a.Dir)
	a.Aw = -a.A * (o.W - target.W) / dist //a.A * math.Sin(a.Dir)
	return a
}

// 计算距离
func (o *Orb) CalcDist(target *Orb) float64 {
	return math.Sqrt((o.X-target.X)*(o.X-target.X) + (o.Y-target.Y)*(o.Y-target.Y) + (o.Z-target.Z)*(o.Z-target.Z) + (o.W-target.W)*(o.W-target.W))
}

func (o *Orb) MarshalJSON() (str []byte, err error) {
	strs := fmt.Sprintf("[%g,%g,%g,%g,%g,%g,%g,%d]", o.X, o.Y, o.Z, o.Vx, o.Vy, o.Vz, o.Mass, o.Id)
	return []byte(strs), nil
}
func (o *Orb) UnmarshalJSON(input []byte) error {
	_, err := fmt.Sscanf(string(input), "[%f,%f,%f,%f,%f,%f,%f,%d]", &o.X, &o.Y, &o.Z, &o.Vx, &o.Vy, &o.Vz, &o.Mass, &o.Id)
	//log.Println("when unmarshal(", string(input), ") n,err,o=", n, err, o)
	return err
}

// 设置撞击 作废
/*
func (o *Orb) SetCrashedBy(crashedBy int) {
	o.crashedBy = crashedBy
}
*/
// 清理orbList中的垃圾
func ClearOrbList(oList []Orb) []Orb {
	allWC = 0
	//var alive int = len(oList)
	for i := 0; i < len(oList); i++ {
		allWC += oList[i].Mass
		if oList[i].Id < 0 {
			oList = append(oList[:i], oList[i+1:]...)
			i--
			//alive--
			//} else {
		}
	}
	//log.Println("when clear alive=", alive)
	clearTimes++
	return oList
}

func ShowMonitorInfo() {
	log.Printf("maxVelo=%.6g %.6g %.6g maxAcc=%.6g %.6g %.6g maxMass=%d %e allMass=%e\n", maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMassId, maxMass, allWC)
}
func GetClearTimes() int64 {
	return clearTimes
}
func GetCrashed() int {
	return nCrashed
}
func GetAllMass() float64 {
	return allMass
}
func GetRealTimes() int64 {
	return realTimes
}
func GetWillTimes() int64 {
	return willTimes
}
