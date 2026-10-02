from sqlalchemy import Column, Integer, String, DateTime
from sqlalchemy.sql import func

from database import Base


class PetProfile(Base):
    """Hồ sơ thú cưng - mỗi thú cưng thuộc về 1 user (chủ nuôi) bên User Service."""

    __tablename__ = "pet_profiles"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), nullable=False)              # tên thú cưng
    species = Column(String(50), nullable=True)              # loài: chó, mèo, ...
    user_id = Column(Integer, nullable=False, index=True)    # id chủ nuôi (User Service)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
