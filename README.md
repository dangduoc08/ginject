# Go web framework inspired by Express JS
Current main features: Routing, middleware, res.JSON

run make mod

tạo 1 đống router tạm khi sử dụng hàm usemiddleware, khi chưa khởi tạo route => hàm merge router cần xử lý thêm để đẩy route vào những router đã tồn tại, thay vì ghi đè

Refactor file:
controller.go => done
router.go => done
merge.go ( remove forEach, add match ) => xong merge router con merge global middleware



Check lai ham format route vs route == ""