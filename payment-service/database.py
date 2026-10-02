from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, declarative_base

# Dùng SQLite để chạy ngay không cần cài đặt DB server.
# Khi nhóm sẵn sàng, chỉ cần đổi dòng dưới sang PostgreSQL, VD:
# DATABASE_URL = "postgresql://user:password@localhost:5432/petprofile_db"
DATABASE_URL = "sqlite:///./petprofile.db"

engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()


def get_db():
    """Dependency cấp 1 session DB cho mỗi request, tự đóng khi xong."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
