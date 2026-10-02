from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class PetProfileBase(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, description="Tên thú cưng")
    species: Optional[str] = Field(None, max_length=50, description="Loài (chó, mèo, ...)")
    user_id: int = Field(..., description="ID chủ nuôi (bên User Service)")


class PetProfileCreate(PetProfileBase):
    pass


class PetProfileUpdate(BaseModel):
    name: Optional[str] = None
    species: Optional[str] = None


class PetProfileOut(PetProfileBase):
    id: int
    created_at: datetime

    class Config:
        from_attributes = True
