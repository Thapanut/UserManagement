package worker

import (
	"context"
	"log"
	"time"

	"backend-challenge/internal/service"
)

// UserCounterWorker ทำหน้าที่รัน Background Goroutine
// เพื่อคอยนับจำนวนผู้ใช้ทั้งหมดในฐานข้อมูลและพิมพ์ Log ออกมาทุกๆ ช่วงเวลาที่กำหนด (ค่าเริ่มต้น 10 วินาที)
type UserCounterWorker struct {
	userService *service.UserService
	interval    time.Duration
}

// NewUserCounterWorker สร้าง Instance ใหม่ของ UserCounterWorker
func NewUserCounterWorker(userService *service.UserService, intervalSeconds int) *UserCounterWorker {
	if intervalSeconds <= 0 {
		intervalSeconds = 10
	}
	return &UserCounterWorker{
		userService: userService,
		interval:    time.Duration(intervalSeconds) * time.Second,
	}
}

// Start เริ่มต้นทำงาน Background Goroutine ตามข้อกำหนด Requirement 6
// และรองรับ Graceful Shutdown ผ่าน context.Context
func (w *UserCounterWorker) Start(ctx context.Context) {
	log.Printf("[BACKGROUND WORKER] Started user counter worker (running every %v)", w.interval)

	// เรียกนับครั้งแรกทันทีเมื่อเริ่มโปรแกรม
	w.logCount(ctx)

	// สร้าง Ticker สำหรับส่งสัญญาณทุกๆ 10 วินาที
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// ได้รับสัญญาณหยุดการทำงาน (Graceful Shutdown)
			log.Println("[BACKGROUND WORKER] Stopping user counter worker...")
			return
		case <-ticker.C:
			// ครบรอบ 10 วินาที ทำการนับจำนวนผู้ใช้ในฐานข้อมูล
			w.logCount(ctx)
		}
	}
}

// logCount ดึงจำนวนผู้ใช้และแสดงผลลง logger
func (w *UserCounterWorker) logCount(ctx context.Context) {
	// ใช้ Timeout สั้นๆ เพื่อไม่ให้ Background Worker ค้างหาก DB ตอบสนองช้า
	countCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	count, err := w.userService.CountUsers(countCtx)
	if err != nil {
		log.Printf("[BACKGROUND WORKER] Error counting users: %v", err)
		return
	}

	log.Printf("[BACKGROUND WORKER] Total users currently registered in database: %d", count)
}
