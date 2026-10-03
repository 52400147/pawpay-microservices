import os

import jwt
from fastapi import Depends, Header, HTTPException

JWT_SECRET = os.environ.get("JWT_SECRET", "pawpay2026_sgu_soa_secretkey")
JWT_ALGORITHM = "HS256"


def get_current_user(authorization: str = Header(None)):
    """Dependency: bắt buộc có Bearer token hợp lệ. Trả về payload {user_id, role}."""
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Chưa đăng nhập hoặc thiếu token")

    token = authorization.split(" ", 1)[1]
    try:
        payload = jwt.decode(token, JWT_SECRET, algorithms=[JWT_ALGORITHM])
    except jwt.ExpiredSignatureError:
        raise HTTPException(status_code=401, detail="Token đã hết hạn")
    except jwt.InvalidTokenError:
        raise HTTPException(status_code=401, detail="Token không hợp lệ")

    return payload


def require_admin(user: dict = Depends(get_current_user)):
    """Dependency: bắt buộc role = admin, dùng cho API thêm/sửa/xóa/liệt kê."""
    if user.get("role") != "admin":
        raise HTTPException(status_code=403, detail="Không đủ quyền truy cập (cần role admin)")
    return user
