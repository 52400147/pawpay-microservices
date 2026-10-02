const express = require('express');
const app = express();
const PORT = 3001;

app.use(express.json());

// Model: User (Chứa các trường: id, họ tên, sdt, email, số dư)
let users = [
    { id: 1, fullName: "Lê Thị Trâm", phone: "0901234567", email: "tram@gmail.com", balance: 500000 },
    { id: 2, fullName: "Nguyen Van A", phone: "0912345678", email: "a@gmail.com", balance: 100000 }
];

// 1. READ ALL - Admin xem danh sách tất cả User
app.get('/admin/users', (req, res) => {
    res.status(200).json({ success: true, data: users });
});

// 2. READ ONE - Admin xem thông tin 1 User theo ID
app.get('/admin/users/:id', (req, res) => {
    const user = users.find(u => u.id === parseInt(req.params.id));
    if (!user) return res.status(404).json({ success: false, message: "Không tìm thấy User" });
    res.status(200).json({ success: true, data: user });
});

// 3. CREATE - Admin thêm mới 1 User
app.post('/admin/users', (req, res) => {
    const { fullName, phone, email, balance } = req.body;
    const newUser = {
        id: users.length ? users[users.length - 1].id + 1 : 1,
        fullName,
        phone,
        email,
        balance: balance || 0
    };
    users.push(newUser);
    res.status(201).json({ success: true, message: "Thêm người dùng thành công", data: newUser });
});

// 4. UPDATE - Admin cập nhật thông tin User
app.put('/admin/users/:id', (req, res) => {
    const { fullName, phone, email, balance } = req.body;
    const userIndex = users.findIndex(u => u.id === parseInt(req.params.id));

    if (userIndex === -1) {
        return res.status(404).json({ success: false, message: "Không tìm thấy User" });
    }

    users[userIndex] = {
        ...users[userIndex],
        fullName: fullName || users[userIndex].fullName,
        phone: phone || users[userIndex].phone,
        email: email || users[userIndex].email,
        balance: balance !== undefined ? balance : users[userIndex].balance
    };

    res.status(200).json({ success: true, message: "Cập nhật thành công", data: users[userIndex] });
});

// 5. DELETE - Admin xóa User
app.delete('/admin/users/:id', (req, res) => {
    const userIndex = users.findIndex(u => u.id === parseInt(req.params.id));
    if (userIndex === -1) {
        return res.status(404).json({ success: false, message: "Không tìm thấy User" });
    }
    const deletedUser = users.splice(userIndex, 1);
    res.status(200).json({ success: true, message: "Xóa người dùng thành công", data: deletedUser });
});

// Chạy server
app.listen(PORT, () => {
    console.log(`User Service đang chạy tại http://localhost:${PORT}`);
});